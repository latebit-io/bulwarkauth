package authentication

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/authentication"
)

type LogonAuthRequest struct {
	TenantID string `json:"tenantId"`
	Email    string `json:"email"`
	Code     string `json:"code"`
	ClientID string `json:"clientId"`
}

func (l LogonAuthRequest) Validate() error {
	if l.TenantID == "" {
		return errors.New("tenant id required")
	}
	if l.Email == "" {
		return errors.New("email required")
	}
	if l.Code == "" {
		return errors.New("code required")
	}
	if l.ClientID == "" {
		return errors.New("client id required")
	}
	return nil
}

type LogonRequest struct {
	TenantID string `json:"tenantId"`
	Email    string `json:"email"`
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

	if err := newLogonRequest.Validate(); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	authenticated, err := h.logonService.Authenticate(c.Request().Context(),
		newLogonRequest.TenantID,
		newLogonRequest.Email, newLogonRequest.ClientID,
		newLogonRequest.Code)
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
	err = h.logonService.Request(c.Request().Context(),
		newLogonRequest.TenantID,
		newLogonRequest.Email)

	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusOK)
}
