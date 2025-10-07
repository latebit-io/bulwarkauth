package authentication

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/authentication/social"
)

type SocialAuthRequest struct {
	ID       string `json:"id" query:"id"`
	Provider string `json:"provider" query:"provider"`
}

type SocialHandlers struct {
	socialService social.SocialService
}

func NewSocialHandlers(socialService social.SocialService) *SocialHandlers {
	return &SocialHandlers{socialService: socialService}
}

func (handler *SocialHandlers) Authenticate(c echo.Context) error {
	socialRequest := new(SocialAuthRequest)
	err := c.Bind(socialRequest)

	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	authenticated, err := handler.socialService.Authenticate(c.Request().Context(), socialRequest.ID,
		socialRequest.Provider)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, authenticated)
}
