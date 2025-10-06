package domain

import "github.com/labstack/echo/v4"

func DomainRoutes(e *echo.Echo, handler *DomainHandlers) {
	e.POST("/api/domain/verify", handler.Verify)
}
