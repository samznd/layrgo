package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samznd/layrgo/internal/fileutil"
	"github.com/samznd/layrgo/internal/templates"
)

var projectDirectories = []string{
	"cmd", "config", "internal", "internal/middleware",
	"internal/models", "internal/repositories", "internal/services",
	"internal/handlers", "internal/routes", "pkg/utils", "scripts",
}

// Scaffold generates a complete project at cfg.ProjectPath and installs its
// dependencies.
func Scaffold(cfg Config) error {
	if err := WriteFiles(cfg); err != nil {
		return err
	}
	return installDependencies(cfg)
}

// WriteFiles generates the project's files and directories without touching the
// network. Separated from Scaffold so tests can cover every stack combination
// without running "go get" for each one.
func WriteFiles(cfg Config) error {
	if cfg.ORM == "" {
		cfg.ORM = "none"
	}
	if err := validateConfig(cfg); err != nil {
		return err
	}

	for _, dir := range projectDirectories {
		fullPath := filepath.Join(cfg.ProjectPath, dir)
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", fullPath, err)
		}
	}

	databaseContent, err := templates.DatabaseFile(cfg.Database, cfg.ORM)
	if err != nil {
		return err
	}

	mainContent, err := templates.MainFile(cfg.ProjectName, cfg.Framework)
	if err != nil {
		return err
	}

	// Ordered slice rather than a map: generation order is then deterministic,
	// so progress output and any failure are reproducible run to run.
	files := []struct {
		rel     string
		content string
	}{
		{filepath.Join("cmd", "main.go"), mainContent},
		{filepath.Join("config", "database.go"), databaseContent},
		{filepath.Join("pkg", "utils", "env_utils.go"), utilsFile()},
		{".env", envFile(cfg.Database)},
		{".env.example", envFile(cfg.Database)},
		{".gitignore", gitignoreFile()},
		{"Dockerfile", dockerfile()},
		{"docker-compose.yml", dockerComposeFile(cfg.Database)},
	}

	for _, f := range files {
		path := filepath.Join(cfg.ProjectPath, f.rel)
		if err := fileutil.CreateFile(path, f.content); err != nil {
			return fmt.Errorf("create %s: %w", f.rel, err)
		}
		logCreated(f.rel)
	}

	return templates.WriteLayers(templates.LayerConfig{
		ProjectPath:   cfg.ProjectPath,
		ProjectName:   cfg.ProjectName,
		Framework:     cfg.Framework,
		ORM:           cfg.ORM,
		OnFileCreated: logCreated,
	})
}

func validateConfig(cfg Config) error {
	if cfg.ProjectName == "" || cfg.ProjectPath == "" {
		return fmt.Errorf("project name and path are required")
	}
	if cfg.Framework == "" || cfg.Database == "" {
		return fmt.Errorf("framework and database are required")
	}
	return nil
}

func envFile(database string) string {
	switch strings.ToLower(database) {
	case "mysql":
		return `DB_USER=root
DB_PASSWORD=password
DB_HOST=localhost
DB_PORT=3306
DB_NAME=mydb`
	case "sqlite":
		return `DB_NAME=mydb.db`
	default:
		return `DB_USER=postgres
DB_PASSWORD=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb`
	}
}

func utilsFile() string {
	return `package utils

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/joho/godotenv"
)

// InitialEnv loads .env for local development. A missing .env is not an error:
// in Docker and other deployments configuration arrives as real environment
// variables, which already take precedence over the file.
func InitialEnv() error {
	if err := godotenv.Load(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load .env file: %w", err)
	}
	return nil
}`
}

func gitignoreFile() string {
	return `# Local environment (contains credentials) - commit .env.example instead
.env

# Build output
/app
/bin/

# SQLite databases
*.db
`
}

func dockerfile() string {
	return `FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app ./cmd

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/app .

EXPOSE 3000
CMD ["./app"]
`
}

func dockerComposeFile(database string) string {
	var (
		dbService string
		dbPort    string
	)

	switch strings.ToLower(database) {
	case "mysql":
		dbPort = "3306"
		dbService = `
  db:
    image: mysql:latest
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: mydb
      MYSQL_USER: user
      MYSQL_PASSWORD: password
    ports:
      - "3306:3306"
    volumes:
      - db-data:/var/lib/mysql
`
	case "sqlite":
		dbPort = ""
		dbService = `
  db:
    image: nouchka/sqlite3
    volumes:
      - db-data:/data
`
	default:
		dbPort = "5432"
		dbService = `
  db:
    image: postgres:latest
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: mydb
    ports:
      - "5432:5432"
    volumes:
      - db-data:/var/lib/postgresql/data
`
	}

	envBlock := `      - DB_HOST=db
      - DB_USER=user
      - DB_PASSWORD=password
      - DB_NAME=mydb`
	if dbPort != "" {
		envBlock += fmt.Sprintf("\n      - DB_PORT=%s", dbPort)
	}

	return fmt.Sprintf(`version: '3.8'

services:
  app:
    build: .
    ports:
      - "3000:3000"
    depends_on:
      - db
    environment:
%s
%s

volumes:
  db-data:
`, envBlock, dbService)
}
