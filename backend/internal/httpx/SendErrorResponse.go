package httpx

import (
	"log"
	"net/http"

	"github.com/ansel1/merry"
)

// SendErrorResponse writes err as a JSON ResponseMessage, using the HTTP
// status code and user message merry carries on it (falling back to 500 /
// "internal server error" for errors that were never merrified with one).
// The raw error is never sent to the client, only logged, since it can
// contain details (SQL, file paths, etc.) that aren't safe to expose.
func SendErrorResponse(responseWriter http.ResponseWriter, err error) {
	statusCode := merry.HTTPCode(err)
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	message := merry.UserMessage(err)
	if message == "" {
		message = "internal server error"
	}
	if statusCode >= http.StatusInternalServerError {
		log.Printf("error: %+v", err)
	}
	SendJSONResponse(responseWriter, statusCode, ResponseMessage{Message: message})
}
