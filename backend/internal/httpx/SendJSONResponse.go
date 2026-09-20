package httpx

import (
	"encoding/json"
	"net/http"
)

func SendJSONResponse(responseWriter http.ResponseWriter, statusCode int, body any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	if body == nil {
		return
	}
	err := json.NewEncoder(responseWriter).Encode(body)
	if err != nil {
		panic(err)
	}
}
