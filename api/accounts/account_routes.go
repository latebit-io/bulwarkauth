package accounts

import "github.com/labstack/echo/v4"

func AccountRoutes(e *echo.Echo, handler AccountHandler) {
	e.POST("/api/accounts", handler.Create)
	e.POST("/api/accounts/verify", handler.Verify)
	e.POST("api/accounts/resend", handler.Resend)
	e.POST("/api/accounts/forgot", handler.Forgot)
	e.POST("/api/accounts/reset", handler.ForgotPassword)
	e.PUT("/api/accounts/delete", handler.DeleteAccount)
	e.PUT("/api/accounts/password", handler.ChangePassword)
	e.PUT("/api/accounts/email", handler.UpdateEmail)
}
