package services

import (
	"errors"
	"tipjar/internal/models"
	"tipjar/internal/repository"
)

type JarService struct {
	jarRepo *repository.JarRepository
	txRepo  *repository.TransactionRepository
}

func NewJarService(jarRepo *repository.JarRepository, txRepo *repository.TransactionRepository) *JarService {
	return &JarService{
		jarRepo: jarRepo,
		txRepo:  txRepo,
	}
}

func (s *JarService) CreateJar(name, description, category string, goalAmount int, deadline *string) (models.Jar, error) {
	// Validate name
	if name == "" {
		return models.Jar{}, errors.New("jar name is required")
	}

	// Validate name length
	if len(name) > 100 {
		return models.Jar{}, errors.New("jar name must be less than 100 characters")
	}

	// Validate goal amount
	if goalAmount < 0 {
		return models.Jar{}, errors.New("goal amount cannot be negative")
	}

	// Validate category
	validCategories := map[string]bool{
		"tip":         true,
		"fundraising": true,
		"event":       true,
		"community":   true,
		"general":     true,
	}

	if category == "" {
		category = "general"
	}

	if !validCategories[category] {
		return models.Jar{}, errors.New("invalid category")
	}

	jar := models.Jar{
		Name:        name,
		Description: description,
		Category:    category,
		GoalAmount:  goalAmount,
	}

	return s.jarRepo.Create(jar)
}

func (s *JarService) GetJar(id string) (models.Jar, []models.Transaction, error) {
	// Validate id
	if id == "" {
		return models.Jar{}, nil, errors.New("jar id is required")
	}

	// Get jar
	jar, err := s.jarRepo.GetByID(id)
	if err != nil {
		return models.Jar{}, nil, errors.New("jar not found")
	}

	// Get completed transactions for this jar
	transactions, err := s.txRepo.GetByJarID(id)
	if err != nil {
		return models.Jar{}, nil, errors.New("failed to get transactions")
	}

	return jar, transactions, nil
}

func (s *JarService) GetProgress(jar models.Jar) int {
	// No goal set
	if jar.GoalAmount == 0 {
		return 0
	}

	progress := (jar.TotalCollected * 100) / jar.GoalAmount

	// Cap at 100
	if progress > 100 {
		return 100
	}

	return progress
}
