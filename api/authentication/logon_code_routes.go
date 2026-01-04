package authentication

import "github.com/labstack/echo/v4"

func LogonRoutes(e *echo.Echo, handler *LogonCodeHandlers, middleware ...echo.MiddlewareFunc) {
	e.POST("/api/authenticate/code", handler.Authenticate, middleware...)
	e.POST("/api/authenticate/logon/request", handler.LogonRequest, middleware...)
}
