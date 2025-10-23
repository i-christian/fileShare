package utils

import (
	"encoding/json"
	"io"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code and payload.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode json response", http.StatusInternalServerError)
	}
}

// ReadJSON decodes a JSON request body into the provided destination struct.
// It also closes the request body automatically.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "unable to read request body", http.StatusBadRequest)
		return err
	}

	if err := json.Unmarshal(body, dst); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return err
	}

	return nil
}

// WriteError returns an error in json format to the client
func WriteErrorJSON(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
