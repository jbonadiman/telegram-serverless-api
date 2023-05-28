package telegram

import (
	"errors"
)

var (
	ErrNoMessagesInRange = errors.New("found no messages in given range")
	ErrChannelIsEmpty    = errors.New("channel is empty")
)
