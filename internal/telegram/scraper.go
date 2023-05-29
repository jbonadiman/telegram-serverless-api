package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

const (
	channelUrlPreview = "https://t.me/s/%s"
	maxMessageCount   = 20
)

const (
	noMessagesSelector     = ".tme_no_messages_found"
	messageSelector        = ".tgme_widget_message_wrap"
	messageDateSelector    = ".tgme_widget_message_date > time"
	messageInfoSelector    = ".tgme_widget_message"
	messageContentSelector = ".tgme_widget_message_text"

	channelImageSelector = ".tgme_channel_info_header img"
	channelNameSelector  = ".tgme_channel_info_header_title"
)

type ScrapeOptions struct {
	Username string
	BeforeID int
	AfterID  int
}

func scrapeMessageDate(outerElement *colly.HTMLElement) (time.Time, error) {
	dateAsText := outerElement.ChildAttr(
		messageDateSelector,
		"datetime",
	)
	parsedDate, err := time.Parse(time.RFC3339, dateAsText)
	if err != nil {
		return time.Time{}, err
	}

	return parsedDate.UTC(), nil
}

func scrapeMessageId(outerElement *colly.HTMLElement) (int, error) {
	idAsText := strings.Split(
		outerElement.ChildAttr(
			messageInfoSelector,
			"data-post",
		), "/",
	)[1]

	parsedId, err := strconv.Atoi(idAsText)
	if err != nil {
		return 0, err
	}

	return parsedId, nil
}

func scrapeMessageContent(outerElement *colly.HTMLElement) string {
	return outerElement.ChildText(messageContentSelector)
}

func scrapeChannelHistory(opt ScrapeOptions) (
	*ChannelHistory,
	error,
) {
	log.Printf("getting messages from %q...\n", opt.Username)

	channelURL := fmt.Sprintf(channelUrlPreview, opt.Username)

	channel := ChannelHistory{
		Username: opt.Username,
		Messages: make([]*Message, 0, maxMessageCount),
	}

	generalError := make([]error, 0, 5)

	c := colly.NewCollector()

	c.OnHTML(noMessagesSelector, func(e *colly.HTMLElement) {
		generalError = append(generalError, ErrNoNewMessages)
	})

	c.OnHTML(
		channelImageSelector, func(e *colly.HTMLElement) {
			channel.ImageURL = e.Attr("src")
		},
	)

	c.OnHTML(
		channelNameSelector, func(e *colly.HTMLElement) {
			channel.Name = e.Text
		},
	)

	c.OnHTML(
		"main", func(e *colly.HTMLElement) {
			e.ForEachWithBreak(
				messageSelector,
				func(_ int, wrapper *colly.HTMLElement) bool {
					parsedDate, err := scrapeMessageDate(wrapper)
					if err != nil {
						generalError = append(generalError, err)
						return false // break
					}

					parsedId, err := scrapeMessageId(wrapper)
					if err != nil {
						generalError = append(generalError, err)
						return false // break
					}

					message := Message{
						Id:      parsedId,
						Date:    parsedDate,
						Content: scrapeMessageContent(wrapper),
					}

					channel.Messages = append(channel.Messages, &message)
					return true
				},
			)
		},
	)

	idFilter := ""

	if opt.AfterID > 0 {
		idFilter = fmt.Sprintf("?after=%d", opt.AfterID)
	} else if opt.BeforeID > 0 {
		idFilter = fmt.Sprintf("?before=%d", opt.BeforeID)
	}

	err := c.Visit(channelURL + idFilter)
	if err != nil {
		return nil, err
	}
	if generalError != nil {
		for _, err := range generalError {
			if err == ErrNoNewMessages {
				return nil, err
			}
		}

		return nil, generalError[len(generalError)-1]
	}

	log.Printf("got %d messages\n", len(channel.Messages))
	return &channel, nil
}
