// Package handlers agrupa handlers HTTP utilizados para testes e exemplo.
package handlers

import (
	"encoding/json"
	"net/http"
)

// TestHandler responde com uma mensagem simples para verificar o limiter.
func TestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Request successful"})
}
