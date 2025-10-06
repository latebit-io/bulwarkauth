package accounts

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauth/api/problem"
	"github.com/latebit-io/bulwarkauth/internal/accounts"
)

type AccountHandler struct {
	accounts accounts.AccountService
}

type NewAccountRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type VerifyAccountRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

type DeleteAccountRequest struct {
	Email       string `json:"email"`
	AccessToken string `json:"accessToken"`
}

type ChangePasswordRequest struct {
	Email       string `json:"email"`
	Password    string `json:"newPassword"`
	AccessToken string `json:"accessToken"`
}

type UpdateEmailRequest struct {
	Email       string `json:"email"`
	AccessToken string `json:"accessToken"`
}

func NewAccountHandler(service accounts.AccountService) AccountHandler {
	return AccountHandler{service}
}

// Create handles the creation of a new account based on the provided email and password in the request payload.
func (ah AccountHandler) Create(c echo.Context) error {
	newAccountRequest := new(NewAccountRequest)
	err := c.Bind(newAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err = ah.accounts.Create(ctx, newAccountRequest.Email, newAccountRequest.Password)
	if err != nil {
		var accountDuplicateError accounts.AccountDuplicateError
		duplicate := errors.As(err, &accountDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Duplicate Account",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

// Verify handles the account verification process using the provided email and verification token in the request payload.
func (ah AccountHandler) Verify(c echo.Context) error {
	verifyAccountRequest := new(VerifyAccountRequest)
	err := c.Bind(verifyAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	err = ah.accounts.Verify(c.Request().Context(), verifyAccountRequest.Email, verifyAccountRequest.Token)
	var accountNotFoundError accounts.AccountNotFoundError
	notFound := errors.As(err, &accountNotFoundError)
	if notFound {
		return echo.NewHTTPError(http.StatusNotFound, problem.Details{
			Type:   "https://latebit.io/bulwark/errors/",
			Title:  "Account not found",
			Status: http.StatusNotFound,
			Detail: err.Error(),
		})
	}

	var verificationError accounts.VerificationError
	notValid := errors.As(err, &verificationError)
	if notValid {
		return echo.NewHTTPError(http.StatusBadRequest, problem.Details{
			Type:   "https://latebit.io/bulwark/errors/",
			Title:  "Verification Error",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah AccountHandler) Resend(c echo.Context) error {
	resendVerificationRequest := new(ResendVerificationRequest)
	err := c.Bind(resendVerificationRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.accounts.Resend(c.Request().Context(), resendVerificationRequest.Email)
	if err == nil {
		return c.NoContent(http.StatusNoContent)
	}

	var accountNotFoundError accounts.AccountNotFoundError
	notFound := errors.As(err, &accountNotFoundError)
	if notFound {
		return echo.NewHTTPError(http.StatusNotFound, problem.Details{
			Type:   "https://latebit.io/bulwark/errors/",
			Title:  "Account not found",
			Status: http.StatusNotFound,
			Detail: err.Error(),
		})
	}

	var verificationError accounts.VerificationError
	notValid := errors.As(err, &verificationError)
	if notValid {
		return echo.NewHTTPError(http.StatusBadRequest, problem.Details{
			Type:   "https://latebit.io/bulwark/errors/",
			Title:  "Verification Error",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	httpError := problem.NewServerError(err)
	return echo.NewHTTPError(httpError.Status, httpError)
}

func (ah AccountHandler) Forgot(c echo.Context) error {
	newForgotPasswordRequest := new(ForgotPasswordRequest)
	err := c.Bind(newForgotPasswordRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.accounts.Forgot(c.Request().Context(), newForgotPasswordRequest.Email)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah AccountHandler) ForgotPassword(c echo.Context) error {
	resetPasswordRequest := new(ResetPasswordRequest)
	err := c.Bind(resetPasswordRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.accounts.ForgotPassword(c.Request().Context(), resetPasswordRequest.Email, resetPasswordRequest.Password, resetPasswordRequest.Token)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah AccountHandler) DeleteAccount(c echo.Context) error {
	deleteAccountRequest := new(DeleteAccountRequest)
	err := c.Bind(deleteAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.accounts.Delete(c.Request().Context(), deleteAccountRequest.Email, deleteAccountRequest.AccessToken)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)

}

func (ah AccountHandler) ChangePassword(c echo.Context) error {
	changePasswordRequest := new(ChangePasswordRequest)
	err := c.Bind(changePasswordRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.accounts.UpdatePassword(c.Request().Context(), changePasswordRequest.Email, changePasswordRequest.Password,
		changePasswordRequest.AccessToken)

	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah AccountHandler) UpdateEmail(c echo.Context) error {
	updateEmailRequest := new(UpdateEmailRequest)
	err := c.Bind(updateEmailRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err = ah.accounts.UpdateEmail(c.Request().Context(), updateEmailRequest.Email, updateEmailRequest.AccessToken)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	return c.NoContent(http.StatusNoContent)
}
