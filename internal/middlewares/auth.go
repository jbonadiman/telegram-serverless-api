package middlewares

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

func Auth(w http.ResponseWriter, r *http.Request) error {
	secret := strings.TrimSpace(os.Getenv("API_TOKEN"))

	if r.Header.Get("Authorization") != "Bearer "+secret {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return errors.New("unauthorized")
	}

	return nil
}
