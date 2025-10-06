package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct{}

func NewHealthHandler() HealthHandler {
	return HealthHandler{}
}

func (h *HealthHandler) Check(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}
