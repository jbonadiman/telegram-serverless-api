package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jbonadiman/telegram-serverless-api/internal"
	"github.com/jbonadiman/telegram-serverless-api/internal/database"
	"github.com/jbonadiman/telegram-serverless-api/internal/telegram"
)

const (
	toDateParamName   = "toDateUTC"
	fromDateParamName = "fromDateUTC"
	channelParam      = "channelId"
)

type apiResponse struct {
	Channel  channelResponse   `json:"channel"`
	Messages []messageResponse `json:"messages"`
}

type channelResponse struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	ImageURL string `json:"image"`
}

type messageResponse struct {
	Id        string `json:"id"`
	DateEpoch int64  `json:"dateEpoch"`
	Content   string `json:"content"`
}

type timeFrame struct {
	FromDate time.Time
	ToDate   time.Time
}

func parseEpoch(epochParam string, paramName string) (time.Time, error) {
	dateEpoch, err := strconv.ParseInt(epochParam, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"%q needs to be a unix epoch",
			paramName,
		)
	}

	return time.Unix(dateEpoch, 0).UTC(), nil
}

func parseQueryParams(queryParams *url.Values) (timeFrame, error) {
	var toDateParsed time.Time

	if !queryParams.Has(fromDateParamName) {
		return timeFrame{}, fmt.Errorf("%q is required", fromDateParamName)
	}

	fromDateParam := queryParams.Get(fromDateParamName)

	fromDateParsed, err := parseEpoch(fromDateParam, fromDateParamName)
	if err != nil {
		return timeFrame{}, err
	}

	if queryParams.Has(toDateParamName) {
		toDateParam := queryParams.Get(toDateParamName)

		toDateParsed, err = parseEpoch(toDateParam, toDateParamName)
		if err != nil {
			return timeFrame{}, err
		}

		if fromDateParsed.After(toDateParsed) {
			return timeFrame{}, fmt.Errorf(
				"%q needs to be before %q",
				fromDateParamName,
				toDateParamName,
			)
		}
	}

	return timeFrame{
		FromDate: fromDateParsed,
		ToDate:   toDateParsed,
	}, nil
}

func initializeDB() telegram.ChannelStorage {
	var db telegram.ChannelStorage
	db, err := database.NewDatabase(internal.DbPath)
	if err != nil {
		log.Fatalln(err.Error())
	}

	return db
}

func Handle(w http.ResponseWriter, r *http.Request) {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	queryParams := r.URL.Query()

	log.Printf("validating query params: %v...\n", queryParams)
	filter, err := parseQueryParams(&queryParams)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	channelUsername := queryParams.Get(channelParam)
	if channelUsername == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("%q is required", channelParam)))
		return
	}

	storage := initializeDB()
	defer storage.Close()

	channel := telegram.NewChannel(
		channelUsername,
		&storage,
	)

	history, err := channel.QueryChannelHistory(telegram.NewQuery(filter.FromDate, filter.ToDate))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	log.Println("parsing response...")
	payload := apiResponse{
		Channel: channelResponse{
			Username: channelUsername,
			Name:     history.Name,
			ImageURL: history.ImageURL,
		},
		Messages: make([]messageResponse, 0, len(history.Messages)),
	}

	for _, msg := range history.Messages {
		payload.Messages = append(
			payload.Messages,
			messageResponse{
				Id:        strconv.Itoa(msg.Id),
				DateEpoch: msg.Date.Unix(),
				Content:   msg.Content,
			},
		)
	}

	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}
