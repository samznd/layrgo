package templates

import "fmt"

func HandlerTemplate(module, framework string) string {
	switch framework {
	case "fiber":
		return fmt.Sprintf(`
package handlers

import (
	"%s/internal/services"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service services.Service
}

func NewHandler(s services.Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Get(c fiber.Ctx) error {
	message, err := h.service.GetMessage()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": message})
}
`, module)
	case "gin":
		return fmt.Sprintf(`
package handlers

import (
	"%s/internal/services"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service services.Service
}

func NewHandler(s services.Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Get(c *gin.Context) {
	message, err := h.service.GetMessage()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": message})
}
`, module)
	case "echo":
		return fmt.Sprintf(`
package handlers

import (
	"net/http"

	"%s/internal/services"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service services.Service
}

func NewHandler(s services.Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Get(c echo.Context) error {
	message, err := h.service.GetMessage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": message})
}
`, module)
	case "chi":
		return fmt.Sprintf(`
package handlers

import (
	"encoding/json"
	"net/http"

	"%s/internal/services"
)

type Handler struct {
	service services.Service
}

func NewHandler(s services.Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	message, err := h.service.GetMessage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
`, module)
	case "iris":
		return fmt.Sprintf(`
package handlers

import (
	"%s/internal/services"

	"github.com/kataras/iris/v12"
)

type Handler struct {
	service services.Service
}

func NewHandler(s services.Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) Get(ctx iris.Context) {
	message, err := h.service.GetMessage()
	if err != nil {
		ctx.StopWithStatus(iris.StatusInternalServerError)
		return
	}
	_ = ctx.JSON(iris.Map{"message": message})
}
`, module)
	default:
		return ""
	}
}

func RoutesTemplate(module, framework string) string {
	switch framework {
	case "fiber":
		return fmt.Sprintf(`
package routes

import (
	"%s/internal/handlers"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(api fiber.Router, h *handlers.Handler) {
	api.Get("/message", h.Get)
}
`, module)
	case "gin":
		return fmt.Sprintf(`
package routes

import (
	"%s/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(api *gin.RouterGroup, h *handlers.Handler) {
	api.GET("/message", h.Get)
}
`, module)
	case "echo":
		return fmt.Sprintf(`
package routes

import (
	"%s/internal/handlers"

	"github.com/labstack/echo/v4"
)

func SetupRoutes(api *echo.Group, h *handlers.Handler) {
	api.GET("/message", h.Get)
}
`, module)
	case "chi":
		return fmt.Sprintf(`
package routes

import (
	"%s/internal/handlers"

	"github.com/go-chi/chi/v5"
)

func SetupRoutes(r chi.Router, h *handlers.Handler) {
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/message", h.Get)
	})
}
`, module)
	case "iris":
		return fmt.Sprintf(`
package routes

import (
	"%s/internal/handlers"

	"github.com/kataras/iris/v12"
)

func SetupRoutes(app *iris.Application, h *handlers.Handler) {
	api := app.Party("/api/v1")
	api.Get("/message", h.Get)
}
`, module)
	default:
		return ""
	}
}
