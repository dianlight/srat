package dto

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dianlight/srat/dm"
)

type Convertible interface {
	From(value interface{}) error
	FromIgnoreEmpty(value interface{}) error
	To(value interface{}) error
	ToIgnoreEmpty(value interface{}) error
	ToResponse(code int, w http.ResponseWriter) error
	ToResponseError(code int, w http.ResponseWriter, message string, body any) error
	FromJSONBody(w http.ResponseWriter, r *http.Request) error
}

// DoResponse writes a JSON response to the provided http.ResponseWriter.
// It sets the HTTP status code and marshals the given body into JSON format.
//
// Parameters:
//   - code: The HTTP status code to be set in the response.
//   - w: The http.ResponseWriter to write the response to.
//   - body: The data to be marshaled into JSON and written as the response body.
//
// If there's an error marshaling the body into JSON, it calls DoResponseError
// with an internal server error status.
func doResponse(code int, w http.ResponseWriter, body any) error {
	w.WriteHeader(code)
	jsonResponse, jsonError := json.Marshal(body)
	if jsonError != nil {
		doResponseError(http.StatusInternalServerError, w, "Unable to encode JSON", jsonError)
		return jsonError
	} else {
		w.Write(jsonResponse)
	}
	return nil
}

// DoResponseError writes a JSON error response to the provided http.ResponseWriter.
// It sets the HTTP status code and marshals an error object into JSON format.
//
// Parameters:
//   - code: The HTTP status code to be set in the response.
//   - w: The http.ResponseWriter to write the response to.
//   - message: A string describing the error message.
//   - body: Additional data to be included in the error response.
//
// The function doesn't return any value. It writes the error response directly to the provided http.ResponseWriter.
// If there's an error marshaling the response into JSON, it writes an internal server error status
// and the error message as plain text.
func doResponseError(code int, w http.ResponseWriter, message string, body any) error {
	w.WriteHeader(code)
	jsonResponse, jsonError := json.Marshal(dm.ResponseError{Error: message, Body: body})
	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
		return jsonError
	} else {
		w.Write(jsonResponse)
	}
	return nil
}

func fromJSONBody(w http.ResponseWriter, r *http.Request, to Convertible) error {
	err := json.NewDecoder(r.Body).Decode(&to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}
