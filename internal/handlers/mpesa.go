package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"tipjar/internal/services"
)

type MpesaHandler struct {
	mpesaService *services.MpesaService
}

func NewMpesaHandler(mpesaService *services.MpesaService) *MpesaHandler {
	return &MpesaHandler{mpesaService: mpesaService}
}

func (h *MpesaHandler) Pay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract jar ID from URL
	// /jars/abc123/pay → abc123
	path := strings.TrimPrefix(r.URL.Path, "/jars/")
	jarID := strings.TrimSuffix(path, "/pay")

	if jarID == "" {
		http.Error(w, "jar id required", http.StatusBadRequest)
		return
	}

	// Read form values
	phone := r.FormValue("phone")
	name := r.FormValue("name")
	message := r.FormValue("message")
	amount := 0
	fmt.Sscanf(r.FormValue("amount"), "%d", &amount)

	// Validate
	if phone == "" {
		http.Error(w, "phone number is required", http.StatusBadRequest)
		return
	}

	if amount < 1 {
		http.Error(w, "amount must be at least 1", http.StatusBadRequest)
		return
	}

	// Format phone number
	phone = formatPhone(phone)

	// Initiate STK push
	tx, err := h.mpesaService.InitiateSTKPush(jarID, phone, amount, name, message)
	if err != nil {
		log.Printf("STK push error: %v", err)
		http.Error(w, "failed to initiate payment", http.StatusInternalServerError)
		return
	}

	// Return checkout ID to frontend for polling
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"checkout_request_id": tx.CheckoutRequestID,
		"message":             "Check your phone for the M-Pesa prompt",
	})
}

func (h *MpesaHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read callback body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("M-Pesa callback received: %s", string(body))

	// Handle callback
	err = h.mpesaService.HandleCallback(body)
	if err != nil {
		log.Printf("Callback handling error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Safaricom expects a 200 response
	// If we return anything else it will retry
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"ResultCode": "0",
		"ResultDesc": "Success",
	})
}

// formatPhone converts local Kenyan numbers to international format
func formatPhone(phone string) string {
	// Remove spaces and dashes
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	if strings.HasPrefix(phone, "0") {
		phone = "254" + phone[1:]
	}

	if strings.HasPrefix(phone, "+") {
		phone = phone[1:]
	}

	return phone
}
