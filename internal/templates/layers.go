package templates

import "fmt"

func RepositoryTemplate(orm string) string {
	switch orm {
	case "gorm":
		return `
package repositories

import (
	"gorm.io/gorm"
)

type Repository interface {
	GetMessage() (string, error)
}

type repoImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repoImpl{db: db}
}

func (r *repoImpl) GetMessage() (string, error) {
	var result struct {
		Message string
	}
	if err := r.db.Raw("SELECT 'data from repository' AS message").Scan(&result).Error; err != nil {
		return "", err
	}
	return result.Message, nil
}
`
	case "xorm":
		return `
package repositories

import (
	"fmt"

	"xorm.io/xorm"
)

type Repository interface {
	GetMessage() (string, error)
}

type repoImpl struct {
	engine *xorm.Engine
}

func NewRepository(engine *xorm.Engine) Repository {
	return &repoImpl{engine: engine}
}

func (r *repoImpl) GetMessage() (string, error) {
	rows, err := r.engine.QueryString("SELECT 'data from repository' AS message")
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no rows returned")
	}
	return rows[0]["message"], nil
}
`
	case "ent":
		return `
package repositories

import (
	"database/sql"
)

// Run ent init to generate schema code before using Ent client types.
type Repository interface {
	GetMessage() (string, error)
}

type repoImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repoImpl{db: db}
}

func (r *repoImpl) GetMessage() (string, error) {
	var message string
	err := r.db.QueryRow("SELECT 'data from repository'").Scan(&message)
	return message, err
}
`
	default:
		return `
package repositories

import "database/sql"

type Repository interface {
	GetMessage() (string, error)
}

type repoImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repoImpl{db: db}
}

func (r *repoImpl) GetMessage() (string, error) {
	var message string
	err := r.db.QueryRow("SELECT 'data from repository'").Scan(&message)
	return message, err
}
`
	}
}

func ServiceTemplate(module string) string {
	return fmt.Sprintf(`
package services

import "%s/internal/repositories"

type Service interface {
	GetMessage() (string, error)
}

type serviceImpl struct {
	repo repositories.Repository
}

func NewService(r repositories.Repository) Service {
	return &serviceImpl{repo: r}
}

func (s *serviceImpl) GetMessage() (string, error) {
	return s.repo.GetMessage()
}
`, module)
}
