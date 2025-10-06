package health

import "github.com/labstack/echo/v4"

func HealthRoutes(e *echo.Echo, handlers HealthHandler) {
	e.GET("/health", handlers.Check)
}
