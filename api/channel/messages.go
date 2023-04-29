package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"telegram_serverless_api/internal/middlewares"
	"telegram_serverless_api/internal/telegram_parser"
)

const (
	toDateParam   = "toDateUTC"
	fromDateParam = "fromDateUTC"
)

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

	return &telegram_parser.Filter{
		ToDate:   toDateParsed,
		FromDate: fromDateParsed,
	}, nil
}

//goland:noinspection GoUnusedExportedFunction
func Handle(w http.ResponseWriter, r *http.Request) {
	err := middlewares.Auth(w, r)
	if err != nil {
		return
	}

	queryParams := r.URL.Query()

	filter, err := parseQueryParams(&queryParams)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	channel := strings.TrimPrefix(r.URL.Path, "/api/channel/messages/")

	parsedMessages, err := telegram_parser.GetChannelMessages(
		channel,
		filter,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	response, err := json.Marshal(parsedMessages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "s-maxage=30, stale-while-revalidate=59")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}
