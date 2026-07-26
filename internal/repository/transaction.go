package repository

import (
	"database/sql"
	"tipjar/internal/models"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(tx models.Transaction) (models.Transaction, error) {
	err := r.db.QueryRow(
		`INSERT INTO transactions (jar_id, phone, name, amount, message, status, checkout_request_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at`,
		tx.JarID,
		tx.Phone,
		tx.Name,
		tx.Amount,
		tx.Message,
		tx.Status,
		tx.CheckoutRequestID,
	).Scan(&tx.ID, &tx.CreatedAt)

	return tx, err
}

func (r *TransactionRepository) GetByCheckoutID(checkoutID string) (models.Transaction, error) {
	var tx models.Transaction

	err := r.db.QueryRow(
		`SELECT id, jar_id, phone, name, amount, message,
		        status, mpesa_receipt, checkout_request_id, created_at
		 FROM transactions
		 WHERE checkout_request_id = $1`,
		checkoutID,
	).Scan(
		&tx.ID,
		&tx.JarID,
		&tx.Phone,
		&tx.Name,
		&tx.Amount,
		&tx.Message,
		&tx.Status,
		&tx.MpesaReceipt,
		&tx.CheckoutRequestID,
		&tx.CreatedAt,
	)

	return tx, err
}

func (r *TransactionRepository) UpdateStatus(checkoutID, status, receipt string) error {
	_, err := r.db.Exec(
		`UPDATE transactions
		 SET status = $1,
		     mpesa_receipt = $2
		 WHERE checkout_request_id = $3`,
		status, receipt, checkoutID,
	)
	return err
}

func (r *TransactionRepository) GetByJarID(jarID string) ([]models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT id, jar_id, phone, name, amount, message,
		        status, mpesa_receipt, checkout_request_id, created_at
		 FROM transactions
		 WHERE jar_id = $1
		 AND status = 'complete'
		 ORDER BY created_at DESC`,
		jarID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.JarID,
			&tx.Phone,
			&tx.Name,
			&tx.Amount,
			&tx.Message,
			&tx.Status,
			&tx.MpesaReceipt,
			&tx.CheckoutRequestID,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}
