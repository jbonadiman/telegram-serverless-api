package database

import (
	"encoding/json"

	"github.com/jbonadiman/telegram-serverless-api/internal/telegram"
	"github.com/tidwall/buntdb"
)

type Database struct {
	db *buntdb.DB
}

func NewDatabase(path string) (*Database, error) {
	db, err := buntdb.Open(path)
	if err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

func (d *Database) SaveHistory(channel *telegram.ChannelHistory) error {
	d.db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get(channel.Username)
		if err == buntdb.ErrNotFound {
			data, err := json.Marshal(channel)
			if err != nil {
				return err
			}
			_, _, err = tx.Set(channel.Username, string(data), nil)
			return err
		}

		if err != nil {
			return err
		}

		var history telegram.ChannelHistory
		if err := json.Unmarshal([]byte(val), &history); err != nil {
			return err
		}

		history.Messages = append(history.Messages, channel.Messages...)

		data, err := json.Marshal(history)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(channel.Username, string(data), nil)

		return err
	})

	return nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) GetHistory(username string) (*telegram.ChannelHistory, error) {
	var history telegram.ChannelHistory

	err := d.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(username)
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(val), &history); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &history, nil
}
