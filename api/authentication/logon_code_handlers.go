package authentication

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/authentication"
)

type LogonAuthRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type LogonRequest struct {
	Email string `json:"email"`
}

type LogonCodeHandlers struct {
	logonService authentication.LogonCodeService
}

func NewLogonCodeHandlers(logonService authentication.LogonCodeService) *LogonCodeHandlers {
	return &LogonCodeHandlers{
		logonService: logonService,
	}
}

func (h *LogonCodeHandlers) Authenticate(c echo.Context) error {
	newLogonRequest := new(LogonAuthRequest)
	err := c.Bind(newLogonRequest)

	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	authenticated, err := h.logonService.Authenticate(c.Request().Context(), newLogonRequest.Email, newLogonRequest.Code)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, authenticated)
}

func (h *LogonCodeHandlers) LogonRequest(c echo.Context) error {
	newLogonRequest := new(LogonRequest)
	err := c.Bind(newLogonRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	err = h.logonService.Request(c.Request().Context(), newLogonRequest.Email)

	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusOK)
}
