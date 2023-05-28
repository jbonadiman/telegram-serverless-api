package internal

import (
	"log"
	"time"
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

type Channel struct {
	Username string
	Storage    *ChannelStore
}

type TelegramChannel interface {
	QueryHistory(channelUsername string, filter Filter) (*ChannelHistory, error)
}

var (
	storage *ChannelStore
)

func NewChannel(channelUsername string) (*Channel, error) {
	if storage == nil {
		storage, err := NewDatabase(DB_PATH)
		if err != nil {
			return nil, err
		}
	}


	return &Channel{
		Username: channelUsername,

	}

func (t *Channel) LoadChannelHistory() error {
	history, err := internal.ScrapeChannelPage(ChannelPageOptions{
		Username: t.ChannelUsername,
	})
	if err != nil {
		return err
	}

	log.Printf("saving %d messages...\n", len(history.Messages))

	var db ChannelStore

	db, err = NewDatabase(DB_PATH)
	if err != nil {
		return err
	}

	return db.SaveHistory(&history)
}

func QueryChannelHistory(channelUsername string, filter *Filter) (
	*ChannelHistory,
	error,
) {
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

	var db ChannelStore

	db, err := NewDatabase(DB_PATH)
	if err != nil {
		return nil, err
	}

	channel, err := db.GetHistory(channelUsername)
	if err != nil {
		return nil, err
	}

	filteredMessages := make([]*Message, 0, len(channel.Messages))

	for _, message := range channel.Messages {
		if message.Date.After(filter.FromDate) &&
			message.Date.Before(filter.ToDate) {
			filteredMessages = append(filteredMessages, message)
		}
	}

	channel.Messages = filteredMessages

	return channel, nil
}
