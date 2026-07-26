package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"tipjar/internal/services"
)

type JarHandler struct {
	jarService *services.JarService
}

func NewJarHandler(jarService *services.JarService) *JarHandler {
	return &JarHandler{jarService: jarService}
}

func (h *JarHandler) CreateJar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	category := r.FormValue("category")
	goalAmount := 0
	fmt.Sscanf(r.FormValue("goal_amount"), "%d", &goalAmount)

	jar, err := h.jarService.CreateJar(name, description, category, goalAmount, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/jars/"+jar.ID, http.StatusSeeOther)
}

func (h *JarHandler) GetJar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/jars/")
	if id == "" {
		http.Error(w, "jar id required", http.StatusBadRequest)
		return
	}

	jar, transactions, err := h.jarService.GetJar(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	progress := h.jarService.GetProgress(jar)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jar":          jar,
		"transactions": transactions,
		"progress":     progress,
	})
}

func (h *JarHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkoutID := r.URL.Query().Get("checkout_id")
	if checkoutID == "" {
		http.Error(w, "checkout_id required", http.StatusBadRequest)
		return
	}

	tx, err := h.jarService.GetTransactionStatus(checkoutID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  tx.Status,
		"receipt": tx.MpesaReceipt,
		"amount":  tx.Amount,
	})
}
func (h *JarHandler) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "templates/home.html")
}
