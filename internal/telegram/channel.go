package telegram

import (
	"log"
	"time"
)

type queryOptions struct {
	FetchAsNeeded bool
	ToDate        time.Time
	FromDate      time.Time
}

type ChannelStorage interface {
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
	storage  ChannelStorage
}

type TelegramChannel interface {
	QueryHistory(string, queryOptions) (*ChannelHistory, error)
}

func NewChannel(channelUsername string, storage *ChannelStorage) *Channel {
	return &Channel{
		Username: channelUsername,
		storage:  *storage,
	}
}

func NewQuery(fromDateUTC, toDateUTC time.Time) *queryOptions {
	return &queryOptions{
		FromDate:      fromDateUTC,
		ToDate:        toDateUTC,
		FetchAsNeeded: true,
	}
}

func (c *Channel) loadChannelHistory(opt ScrapeOptions) error {
	history, err := scrapeChannelHistory(opt)
	if err != nil {
		return err
	}

	log.Printf("saving %d messages...\n", len(history.Messages))

	return c.storage.SaveHistory(history)
}

func (c *Channel) QueryChannelHistory(opt *queryOptions) (*ChannelHistory, error) {
	if opt.FromDate.IsZero() {
		opt.FromDate = time.Unix(0, 0)
	}

	if opt.ToDate.IsZero() {
		opt.ToDate = time.Now().UTC()
	}

	channel, err := c.storage.GetHistory(c.Username)
	if err != nil && err != ErrChannelIsEmpty {
		return nil, err
	}

	log.Printf(
		"filtering messages from %s to %s...\n",
		opt.FromDate.Format("2006-01-02"),
		opt.ToDate.Format("2006-01-02"),
	)

	if err == ErrChannelIsEmpty {
		if opt.FetchAsNeeded {
			log.Println("channel is empty, fetching history...")
			err = c.loadChannelHistory(ScrapeOptions{
				Username: c.Username,
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, ErrNoMessagesInRange
		}
	}

	for {
		lastMessage := channel.Messages[len(channel.Messages)-1]

		if lastMessage.Date.After(opt.ToDate) {
			break
		}

		// needs to update local storage, fetching new messages
		if !opt.FetchAsNeeded {
			return nil, ErrNoMessagesInRange
		}

		log.Println("fetching newer history...")
		err = c.loadChannelHistory(ScrapeOptions{
			Username: c.Username,
			AfterID:  lastMessage.Id,
		})
		if err == ErrNoNewMessages {
			log.Println("found no new messages")
			break
		}
		if err != nil {
			return nil, err
		}

		channel, err = c.storage.GetHistory(c.Username)
		if err != nil {
			return nil, err
		}
	}

	filteredMessages := make([]*Message, 0, len(channel.Messages))

	for _, message := range channel.Messages {
		if message.Date.After(opt.FromDate) &&
			message.Date.Before(opt.ToDate) {
			filteredMessages = append(filteredMessages, message)
		}
	}

	channel.Messages = filteredMessages

	return channel, nil
}
