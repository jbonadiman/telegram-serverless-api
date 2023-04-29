package telegram_serverless_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"telegram_serverless_api/internal/telegram_parser"
)

const (
	toDateParam   = "toDateUTC"
	fromDateParam = "fromDateUTC"
	channelParam  = "channelId"
)

type apiResponse struct {
	ChannelId string           `json:"channel"`
	Messages  []channelMessage `json:"messages"`
}

type channelMessage struct {
	Id        string `json:"id"`
	Image     string `json:"image"`
	DateEpoch int64  `json:"dateEpoch"`
	Content   string `json:"content"`
}

func validateDateParam(param string, queryParams *url.Values) (
	time.Time,
	error,
) {
	if !queryParams.Has(param) {
		return time.Time{}, errors.New(fmt.Sprintf("%q is required", param))
	}

	dateEpoch, err := strconv.ParseInt(queryParams.Get(param), 10, 64)
	if err != nil {
		return time.Time{}, errors.New(
			fmt.Sprintf(
				"%q needs to be a unix epoch",
				param,
			),
		)
	}

	return time.Unix(dateEpoch, 0).UTC(), nil
}

func parseQueryParams(queryParams *url.Values) (
	*telegram_parser.Filter,
	error,
) {
	toDateParsed, err := validateDateParam(toDateParam, queryParams)
	if err != nil {
		return nil, err
	}

	fromDateParsed, err := validateDateParam(fromDateParam, queryParams)
	if err != nil {
		return nil, err
	}

	if fromDateParsed.After(toDateParsed) {
		return nil, errors.New(
			fmt.Sprintf(
				"%q needs to be before %q",
				fromDateParam,
				toDateParam,
			),
		)
	}

	return &telegram_parser.Filter{
		ToDate:   toDateParsed,
		FromDate: fromDateParsed,
	}, nil
}

//goland:noinspection GoUnusedExportedFunction
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

	channel := queryParams.Get(channelParam)
	if channel == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("%q is required", channelParam)))
		return
	}

	parsedMessages, err := telegram_parser.GetChannelMessages(
		channel,
		filter,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	log.Println("parsing response...")
	payload := apiResponse{
		ChannelId: channel,
		Messages:  make([]channelMessage, 0, len(parsedMessages)),
	}

	for _, message := range parsedMessages {
		payload.Messages = append(
			payload.Messages,
			channelMessage{
				Id:        strconv.Itoa(message.Id),
				Image:     message.Image.String(),
				DateEpoch: message.Date.Unix(),
				Content:   message.Content,
			},
		)
	}

	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "s-maxage=10, stale-while-revalidate=59")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}
