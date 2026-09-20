package handler

import (
	"encoding/json"
	"net/http"
)

type pingResponse struct {
	Message string `json:"message"`
}

// Give reachability signal
func Ping(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pingResponse{Message: "PONG!"})
}
