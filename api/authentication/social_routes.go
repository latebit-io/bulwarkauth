package authentication

import "github.com/labstack/echo/v4"

func SocialRoutes(e *echo.Echo, handler *SocialHandlers) {
	e.POST("/api/authenticate/social", handler.Authenticate)
}
