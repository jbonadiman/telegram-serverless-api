package internal

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

const (
	channelURLPreview = "https://t.me/s/%s"
	maxMessageCount   = 20
)

const (
	messageSelector        = ".tgme_widget_message_wrap"
	messageDateSelector    = ".tgme_widget_message_date > time"
	messageInfoSelector    = ".tgme_widget_message"
	messageContentSelector = ".tgme_widget_message_text"

	channelImageSelector = ".tgme_channel_info_header img"
	channelNameSelector  = ".tgme_channel_info_header_title"
)

type Filter struct {
	ToDate   time.Time
	FromDate time.Time
}

type ChannelHistory struct {
	Username string
	Name     string
	ImageURL string
	Messages []*Message
}

type Message struct {
	Id      int
	Date    time.Time
	Content string
}

func getMessageDate(outerElement *colly.HTMLElement) (time.Time, error) {
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

func getMessageId(outerElement *colly.HTMLElement) (int, error) {
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

func getMessageContent(outerElement *colly.HTMLElement) string {
	return outerElement.ChildText(messageContentSelector)
}

func GetChannelMessages(channelUsername string, filter *Filter) (
	*ChannelHistory,
	error,
) {
	log.Printf("getting messages from %q...\n", channelUsername)

	if filter.FromDate.IsZero() {
		filter.FromDate = time.Unix(0, 0)
	}

	if filter.ToDate.IsZero() {
		filter.ToDate = time.Now().UTC()
	}

	log.Printf(
		"filtering messages from %s to %s...\n",
		filter.FromDate.Format("2006-01-02"),
		filter.ToDate.Format("2006-01-02"),
	)

	channelURL := fmt.Sprintf(channelURLPreview, channelUsername)

	channel := ChannelHistory{
		Username: channelUsername,
		Messages: make([]*Message, 0, maxMessageCount),
	}

	var generalError *error

	c := colly.NewCollector()

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
					parsedDate, err := getMessageDate(wrapper)
					if err != nil {
						generalError = &err
						return false // break
					}

					if parsedDate.Before(filter.FromDate) || parsedDate.After(filter.ToDate) {
						return true // continue
					}

					parsedId, err := getMessageId(wrapper)
					if err != nil {
						generalError = &err
						return false // break
					}

					message := Message{
						Id:      parsedId,
						Date:    parsedDate,
						Content: getMessageContent(wrapper),
					}

					channel.Messages = append(channel.Messages, &message)
					return true
				},
			)
		},
	)

	err := c.Visit(channelURL)
	if err != nil {
		return nil, err
	}
	if generalError != nil {
		return nil, *generalError
	}

	log.Printf("got %d messages\n", len(channel.Messages))
	return &channel, nil
}
