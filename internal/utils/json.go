package utils

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code and payload.
func WriteJSON(w http.ResponseWriter, status int, data any, headers http.Header, logger *slog.Logger) {
	if data == nil {
		return
	}

	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		logger.Error(err.Error())
		http.Error(w, "failed to encode json response", http.StatusInternalServerError)
		return
	}

	js = append(js, '\n')
	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
}

// ReadJSON decodes a JSON request body into the provided destination struct.
// It also closes the request body automatically.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(&dst); err != nil {
		return err
	}

	return nil
}

// WriteError returns an error in json format to the client
func WriteErrorJSON(w http.ResponseWriter, status int, msg string, logger *slog.Logger) {
	WriteJSON(w, status, map[string]string{"error": msg}, nil, logger)
}
