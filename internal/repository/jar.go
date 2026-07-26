package repository

import (
	"database/sql"
	"tipjar/internal/models"
)

type JarRepository struct {
	db *sql.DB
}

func NewJarRepository(db *sql.DB) *JarRepository {
	return &JarRepository{db: db}
}

func (r *JarRepository) Create(jar models.Jar) (models.Jar, error) {
	err := r.db.QueryRow(
		`INSERT INTO jars (name, description, category, goal_amount, deadline)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		jar.Name,
		jar.Description,
		jar.Category,
		jar.GoalAmount,
		jar.Deadline,
	).Scan(&jar.ID, &jar.CreatedAt)

	return jar, err
}

func (r *JarRepository) GetByID(id string) (models.Jar, error) {
	var jar models.Jar

	err := r.db.QueryRow(
		`SELECT id, name, description, category, goal_amount, 
		        total_collected, deadline, created_at
		 FROM jars
		 WHERE id = $1`,
		id,
	).Scan(
		&jar.ID,
		&jar.Name,
		&jar.Description,
		&jar.Category,
		&jar.GoalAmount,
		&jar.TotalCollected,
		&jar.Deadline,
		&jar.CreatedAt,
	)

	return jar, err
}

func (r *JarRepository) UpdateTotal(jarID string, amount int) error {
	_, err := r.db.Exec(
		`UPDATE jars 
		 SET total_collected = total_collected + $1
		 WHERE id = $2`,
		amount, jarID,
	)
	return err
}
