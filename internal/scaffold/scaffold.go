package scaffold

import (
	"fmt"
	"strings"

	"github.com/samznd/layrgo/internal/fileutil"
	"github.com/samznd/layrgo/internal/templates"
)

var projectDirectories = []string{
	"cmd", "config", "internal", "internal/middleware",
	"internal/models", "internal/repositories", "internal/services",
	"internal/handlers", "internal/routes", "pkg/utils", "scripts",
}

// Scaffold generates a complete project at cfg.ProjectPath.
func Scaffold(cfg Config) error {
	if cfg.ORM == "" {
		cfg.ORM = "none"
	}
	if err := validateConfig(cfg); err != nil {
		return err
	}

	for _, dir := range projectDirectories {
		fullPath := joinPath(cfg.ProjectPath, dir)
		if err := mkdirAll(fullPath); err != nil {
			return fmt.Errorf("create directory %s: %w", fullPath, err)
		}
	}

	databaseContent := databaseFile(cfg.Database, cfg.ORM)
	if databaseContent == "" {
		return fmt.Errorf("invalid database configuration: database=%s orm=%s", cfg.Database, cfg.ORM)
	}

	files := map[string]string{
		joinPath(cfg.ProjectPath, "cmd", "main.go"):            mainFile(cfg),
		joinPath(cfg.ProjectPath, "config", "database.go"):     databaseContent,
		joinPath(cfg.ProjectPath, ".env"):                      envFile(cfg.Database),
		joinPath(cfg.ProjectPath, "pkg/utils", "env_utils.go"): utilsFile(),
		joinPath(cfg.ProjectPath, "Dockerfile"):                dockerfile(),
		joinPath(cfg.ProjectPath, "docker-compose.yml"):        dockerComposeFile(cfg.Database),
	}

	for path, content := range files {
		if err := fileutil.CreateFile(path, content); err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
	}

	if err := templates.WriteLayers(templates.LayerConfig{
		ProjectPath: cfg.ProjectPath,
		ProjectName: cfg.ProjectName,
		Framework:   cfg.Framework,
		ORM:         cfg.ORM,
	}); err != nil {
		return err
	}

	return installDependencies(cfg)
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

func diBlock() string {
	return `	repo := repositories.NewRepository(config.DB)
	service := services.NewService(repo)
	handler := handlers.NewHandler(service)`
}

func mainFile(cfg Config) string {
	mod := cfg.ProjectName
	di := diBlock()

	switch strings.ToLower(cfg.Framework) {
	case "fiber":
		return fmt.Sprintf(`package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"%s/config"
	"%s/internal/handlers"
	"%s/internal/repositories"
	"%s/internal/routes"
	"%s/internal/services"
	"%s/pkg/utils"
)

func main() {
	if err := utils.InitialEnv(); err != nil {
		log.Fatal(err)
	}
	config.Connect()

	app := fiber.New()

%s

	api := app.Group("/api/v1")
	routes.SetupRoutes(api, handler)

	log.Println("🚀 Fiber server is running on http://localhost:3000")
	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}`, mod, mod, mod, mod, mod, mod, di)
	case "gin":
		return fmt.Sprintf(`package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"%s/config"
	"%s/internal/handlers"
	"%s/internal/repositories"
	"%s/internal/routes"
	"%s/internal/services"
	"%s/pkg/utils"
)

func main() {
	if err := utils.InitialEnv(); err != nil {
		log.Fatal(err)
	}
	config.Connect()

	r := gin.Default()

%s

	api := r.Group("/api/v1")
	routes.SetupRoutes(api, handler)

	log.Println("🚀 Gin server is running on http://localhost:3000")
	if err := r.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}`, mod, mod, mod, mod, mod, mod, di)
	case "echo":
		return fmt.Sprintf(`package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"%s/config"
	"%s/internal/handlers"
	"%s/internal/repositories"
	"%s/internal/routes"
	"%s/internal/services"
	"%s/pkg/utils"
)

func main() {
	if err := utils.InitialEnv(); err != nil {
		log.Fatal(err)
	}
	config.Connect()

	e := echo.New()

%s

	api := e.Group("/api/v1")
	routes.SetupRoutes(api, handler)

	log.Println("🚀 Echo server is running on http://localhost:3000")
	if err := e.Start(":3000"); err != nil {
		log.Fatal(err)
	}
}`, mod, mod, mod, mod, mod, mod, di)
	case "chi":
		return fmt.Sprintf(`package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"%s/config"
	"%s/internal/handlers"
	"%s/internal/repositories"
	"%s/internal/routes"
	"%s/internal/services"
	"%s/pkg/utils"
)

func main() {
	if err := utils.InitialEnv(); err != nil {
		log.Fatal(err)
	}
	config.Connect()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

%s

	routes.SetupRoutes(r, handler)

	log.Println("🚀 Chi server is running on http://localhost:3000")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}`, mod, mod, mod, mod, mod, mod, di)
	case "iris":
		return fmt.Sprintf(`package main

import (
	"log"

	"github.com/kataras/iris/v12"
	"%s/config"
	"%s/internal/handlers"
	"%s/internal/repositories"
	"%s/internal/routes"
	"%s/internal/services"
	"%s/pkg/utils"
)

func main() {
	if err := utils.InitialEnv(); err != nil {
		log.Fatal(err)
	}
	config.Connect()

	app := iris.New()

%s

	routes.SetupRoutes(app, handler)

	log.Println("🚀 Iris server is running on http://localhost:3000")
	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}`, mod, mod, mod, mod, mod, mod, di)
	default:
		return ""
	}
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
	"fmt"

	"github.com/joho/godotenv"
)

func InitialEnv() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("load .env file: %w", err)
	}
	return nil
}`
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

	switch database {
	case "MySQL":
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
	case "SQLite":
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

	envBlock := fmt.Sprintf(`      - DB_HOST=db
      - DB_USER=user
      - DB_PASSWORD=password
      - DB_NAME=mydb`)
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
