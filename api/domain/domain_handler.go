package domain

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/domain"
)

type DomainHandlers struct {
	domainService domain.DomainService
}

func NewDomainHandler(domainService domain.DomainService) *DomainHandlers {
	return &DomainHandlers{
		domainService: domainService,
	}
}

func (d *DomainHandlers) Verify(c echo.Context) error {
	domains, err := d.domainService.GetAll(c.Request().Context())
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	for _, dm := range domains {
		err := d.domainService.Verify(c.Request().Context(), dm.Domain)
		if err != nil {
			httpError := problem.NewBadRequest(err)
			return echo.NewHTTPError(httpError.Status, httpError)
		}
	}
	return c.NoContent(http.StatusOK)
}
