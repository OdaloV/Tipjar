package models

import "time"

type Jar struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	GoalAmount     int        `json:"goal_amount"`
	TotalCollected int        `json:"total_collected"`
	Deadline       *time.Time `json:"deadline"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Transaction struct {
	ID                string    `json:"id"`
	JarID             string    `json:"jar_id"`
	Phone             string    `json:"phone"`
	Name              string    `json:"name"`
	Amount            int       `json:"amount"`
	Message           string    `json:"message"`
	Status            string    `json:"status"`
	MpesaReceipt      string    `json:"mpesa_receipt"`
	CheckoutRequestID string    `json:"checkout_request_id"`
	CreatedAt         time.Time `json:"created_at"`
}
