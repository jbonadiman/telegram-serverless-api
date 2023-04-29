package telegram_parser

import (
	"fmt"
	"log"
	"net/url"
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
	historySelector     = ".tgme_channel_history"
	messagesSelector    = ".tgme_widget_message_wrap"
	dateSelector        = ".tgme_widget_message_date > time"
	messageInfoSelector = ".tgme_widget_message"
	imageSelector       = ".tgme_widget_message_user_photo > img"
	contentSelector     = ".tgme_widget_message_text"
)

type Filter struct {
	ToDate   time.Time
	FromDate time.Time
}

type Message struct {
	Id      int       `json:"id"`
	Image   url.URL   `json:"image"`
	Date    time.Time `json:"date"`
	Content string    `json:"content"`
}

func getMessageDate(outerElement *colly.HTMLElement) (time.Time, error) {
	dateAsText := outerElement.ChildAttr(
		dateSelector,
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

func getMessageImage(outerElement *colly.HTMLElement) (*url.URL, error) {
	parsedUrl, err := url.Parse(
		outerElement.ChildAttr(
			imageSelector,
			"src",
		),
	)
	if err != nil {
		return nil, err
	}

	return parsedUrl, nil
}

func getMessageContent(outerElement *colly.HTMLElement) string {
	return outerElement.ChildText(contentSelector)
}

func GetChannelMessages(channelUsername string, filter *Filter) (
	[]Message,
	error,
) {
	log.Printf("getting messages from %q...\n", channelUsername)

	if filter.FromDate.IsZero() {
		filter.FromDate = time.Unix(0, 0)
	}

	if filter.ToDate.IsZero() {
		filter.ToDate = time.Now().UTC()
	}

	if filter.FromDate.After(filter.ToDate) {
		return nil, fmt.Errorf(
			"cannot search from messages backwards: fromDate %q is after toDate %q",
			filter.FromDate.Format("2006-01-02"),
			filter.ToDate.Format("2006-01-02"),
		)
	}

	log.Printf(
		"filtering messages from %s to %s...\n",
		filter.FromDate.Format("2006-01-02"),
		filter.ToDate.Format("2006-01-02"),
	)

	channelURL := fmt.Sprintf(channelURLPreview, channelUsername)

	messageList := make([]Message, 0, maxMessageCount)
	var generalError *error

	c := colly.NewCollector()

	c.OnHTML(
		historySelector, func(history *colly.HTMLElement) {
			history.ForEachWithBreak(
				messagesSelector,
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

					parsedUrl, err := getMessageImage(wrapper)
					if err != nil {
						generalError = &err
						return false // break
					}

					message := Message{
						Id:      parsedId,
						Image:   *parsedUrl,
						Date:    parsedDate,
						Content: getMessageContent(wrapper),
					}

					messageList = append(messageList, message)
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

	log.Printf("got %d messages\n", len(messageList))
	return messageList, nil
}
