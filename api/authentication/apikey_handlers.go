package authentication

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/authentication"
)

type CreateApiKeyRequest struct {
	TenantID    string     `json:"tenantId"`
	AccessToken string     `json:"accessToken"`
	Name        string     `json:"name"`
	Expires     *time.Time `json:"expires"`
}

func (r CreateApiKeyRequest) Validate() error {
	if r.TenantID == "" {
		return errors.New("tenantId required")
	}
	if r.AccessToken == "" {
		return errors.New("accessToken required")
	}
	if r.Name == "" {
		return errors.New("name required")
	}
	return nil
}

type ListApiKeyRequest struct {
	TenantID    string `json:"tenantId"`
	AccessToken string `json:"accessToken"`
}

func (r ListApiKeyRequest) Validate() error {
	if r.TenantID == "" {
		return errors.New("tenantId required")
	}
	if r.AccessToken == "" {
		return errors.New("accessToken required")
	}
	return nil
}

type DeleteApiKeyRequest struct {
	TenantID    string `json:"tenantId"`
	AccessToken string `json:"accessToken"`
}

func (r DeleteApiKeyRequest) Validate() error {
	if r.TenantID == "" {
		return errors.New("tenantId required")
	}
	if r.AccessToken == "" {
		return errors.New("accessToken required")
	}
	return nil
}

type ApiKeyHandlers struct {
	apiKeyService authentication.ApiKeyService
}

func NewApiKeyHandlers(service authentication.ApiKeyService) *ApiKeyHandlers {
	return &ApiKeyHandlers{apiKeyService: service}
}

func (h *ApiKeyHandlers) Create(c echo.Context) error {
	req := new(CreateApiKeyRequest)
	err := c.Bind(req)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	if err := req.Validate(); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	created, err := h.apiKeyService.Create(c.Request().Context(), req.TenantID, req.AccessToken, req.Name, req.Expires)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusCreated, created)
}

func (h *ApiKeyHandlers) List(c echo.Context) error {
	req := new(ListApiKeyRequest)
	err := c.Bind(req)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	if err := req.Validate(); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	keys, err := h.apiKeyService.List(c.Request().Context(), req.TenantID, req.AccessToken)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, keys)
}

func (h *ApiKeyHandlers) Delete(c echo.Context) error {
	req := new(DeleteApiKeyRequest)
	err := c.Bind(req)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	if err := req.Validate(); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	prefix := c.Param("prefix")
	if prefix == "" {
		httpError := problem.NewBadRequest(errors.New("prefix required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = h.apiKeyService.Delete(c.Request().Context(), req.TenantID, req.AccessToken, prefix)
	if err != nil {
		var notFound authentication.ApiKeyNotFoundError
		if errors.As(err, &notFound) {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "API Key Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}
