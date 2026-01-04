package accounts

import "github.com/labstack/echo/v4"

func AccountRoutes(e *echo.Echo, handler AccountHandler, middleware ...echo.MiddlewareFunc) {
	e.POST("/api/accounts", handler.Register, middleware...)
	e.POST("/api/accounts/verify", handler.Verify)
	e.POST("api/accounts/resend", handler.Resend, middleware...)
	e.POST("/api/accounts/forgot", handler.Forgot, middleware...)
	e.POST("/api/accounts/reset", handler.ForgotPassword, middleware...)
	e.PUT("/api/accounts/delete", handler.DeleteAccount)
	e.PUT("/api/accounts/password", handler.ChangePassword)
	e.PUT("/api/accounts/email", handler.UpdateEmail)
}
