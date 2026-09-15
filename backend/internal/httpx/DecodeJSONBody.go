package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/ansel1/merry"
)

func DecodeJSONBody[T any](request *http.Request) (T, error) {
	var body T
	err := json.NewDecoder(request.Body).Decode(&body)
	if err != nil {
		return body, merry.Wrap(err).WithHTTPCode(http.StatusBadRequest).WithUserMessage("invalid request body")
	}
	return body, nil
}
