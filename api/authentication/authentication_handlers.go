package authentication

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/authentication"
)

type AuthenticationRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RenewRequest struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refreshToken"`
}

type AcknowledgeRequest struct {
	Email        string `json:"email"`
	ClientId     string `json:"clientId"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type RevokeRequest struct {
	Email       string `json:"email"`
	ClientId    string `json:"clientId"`
	AccessToken string `json:"accessToken"`
}

type ValidateAccessTokenRequest struct {
	Email    string `json:"email"`
	ClientId string `json:"clientId"`
	Token    string `json:"token"`
}

type AuthenticationHandler struct {
	authentication authentication.AuthenticationService
}

func NewAuthenticationHandler(service authentication.AuthenticationService) AuthenticationHandler {
	return AuthenticationHandler{service}
}

func (ah AuthenticationHandler) Authenticate(c echo.Context) error {
	newAuthRequest := new(AuthenticationRequest)
	err := c.Bind(newAuthRequest)

	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	authenticated, err := ah.authentication.Authenticate(c.Request().Context(), newAuthRequest.Email, newAuthRequest.Password)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, authenticated)
}

func (ah AuthenticationHandler) Acknowledge(c echo.Context) error {
	newAckRequest := new(AcknowledgeRequest)
	err := c.Bind(newAckRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	_, err = ah.authentication.ValidateAccessToken(c.Request().Context(), newAckRequest.AccessToken, newAckRequest.Email)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	_, err = ah.authentication.ValidateRefreshToken(c.Request().Context(), newAckRequest.RefreshToken, newAckRequest.Email)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.authentication.Acknowledge(c.Request().Context(), authentication.Authenticated{
		AccessToken:  newAckRequest.AccessToken,
		RefreshToken: newAckRequest.RefreshToken,
	}, newAckRequest.Email, newAckRequest.ClientId)

	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

func (ah AuthenticationHandler) Renew(c echo.Context) error {
	newRenewRequest := new(RenewRequest)
	err := c.Bind(newRenewRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	authenticated, err := ah.authentication.Renew(c.Request().Context(), newRenewRequest.RefreshToken, newRenewRequest.Email)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, authenticated)
}

func (ah AuthenticationHandler) Revoke(c echo.Context) error {
	newRevokeRequest := new(RevokeRequest)
	if err := c.Bind(newRevokeRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	_, err := ah.authentication.ValidateAccessToken(c.Request().Context(), newRevokeRequest.AccessToken, newRevokeRequest.Email)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.authentication.Revoke(c.Request().Context(), newRevokeRequest.ClientId, newRevokeRequest.Email, newRevokeRequest.AccessToken)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return nil
}

func (ah AuthenticationHandler) ValidateAccessToken(c echo.Context) error {
	validateAccessTokenRequest := new(ValidateAccessTokenRequest)
	if err := c.Bind(validateAccessTokenRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	claims, err := ah.authentication.ValidateAccessToken(c.Request().Context(), validateAccessTokenRequest.Token, validateAccessTokenRequest.Email)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	return c.JSON(http.StatusOK, claims)
}
