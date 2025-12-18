package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"distributed-rate-limiter/internal/limiter"
	"distributed-rate-limiter/internal/models"
)

func main() {
	rl := limiter.NewFixedWindowLimiter(5, time.Minute)

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.CheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Key == "" {
			http.Error(w, "key is required", http.StatusBadRequest)
			return
		}

		allowed, err := rl.Allow(r.Context(), req.Key)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := models.CheckResponse{Allowed: allowed}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
