package authentication

import "github.com/labstack/echo/v4"

func LogonRoutes(e *echo.Echo, handler *LogonCodeHandlers) {
	e.POST("/api/authenticate/code", handler.Authenticate)
	e.POST("/api/authenticate/logon/request", handler.LogonRequest)
}
