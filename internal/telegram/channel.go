package telegram

import (
	"log"
	"time"
)

type Filter struct {
	ToDate   time.Time
	FromDate time.Time
}

type ChannelStore interface {
	SaveHistory(channel *ChannelHistory) error
	GetHistory(username string) (*ChannelHistory, error)

	Close() error
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
	storage  ChannelStore
}

type TelegramChannel interface {
	QueryHistory(channelUsername string, filter Filter) (*ChannelHistory, error)
}

func NewChannel(channelUsername string, storage *ChannelStore) (*Channel, error) {
	return &Channel{
		Username: channelUsername,
		storage:  *storage,
	}, nil
}

func (c *Channel) LoadChannelHistory() error {
	history, err := ScrapeChannelHistory(ScrapeOptions{
		Username: c.Username,
	})
	if err != nil {
		return err
	}

	log.Printf("saving %d messages...\n", len(history.Messages))

	return c.storage.SaveHistory(history)
}

func (c *Channel) QueryChannelHistory(filter *Filter) (*ChannelHistory, error) {
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

	channel, err := c.storage.GetHistory(c.Username)
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
