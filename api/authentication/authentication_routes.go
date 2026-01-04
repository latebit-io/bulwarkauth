package authentication

import "github.com/labstack/echo/v4"

func AuthenticationRoutes(e *echo.Echo, handler AuthenticationHandler, middleware ...echo.MiddlewareFunc) {
	e.POST("/api/authenticate", handler.Authenticate, middleware...)
	e.POST("/api/authenticate/ack", handler.Acknowledge)
	e.POST("/api/authenticate/renew", handler.Renew, middleware...)
	e.DELETE("/api/authenticate/revoke", handler.Revoke)
	e.POST("/api/authenticate/token/validate", handler.ValidateAccessToken)
}
