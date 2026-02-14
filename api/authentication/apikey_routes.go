package authentication

import "github.com/labstack/echo/v4"

func ApiKeyRoutes(e *echo.Echo, handler *ApiKeyHandlers, middleware ...echo.MiddlewareFunc) {
	e.POST("/api/apikeys", handler.Create, middleware...)
	e.POST("/api/apikeys/list", handler.List, middleware...)
	e.DELETE("/api/apikeys/:prefix", handler.Delete, middleware...)
}
