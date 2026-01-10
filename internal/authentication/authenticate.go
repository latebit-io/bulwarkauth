package authentication

import (
	"context"
	"errors"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"github.com/latebit-io/bulwarkauth/internal/utils"
)

// AuthenticationService defines the interface for authentication services.
type AuthenticationService interface {
	Authenticate(ctx context.Context, tenantID, email, clientID, password string) (*Authenticated, error)
	Acknowledge(ctx context.Context, tenantID string, Authenticate Authenticated) error
	ValidateAccessToken(ctx context.Context, tenantID, accessToken string) (*AccessTokenClaims, error)
	ValidateRefreshToken(ctx context.Context, tenantID, refreshToken string) (*RefreshTokenClaims, error)
	Renew(ctx context.Context, tenantID, refreshToken string) (*Authenticated, error)
	Revoke(ctx context.Context, tenantID, accessToken string) error
}

type AccountRepository interface {
	Read(ctx context.Context, tenantID, email string) (*accounts.Account, error)
	PasswordMatches(ctx context.Context, tenantID, email, password string) (bool, error)
}

type Tokenizer interface {
	CreateAccessToken(ctx context.Context, tenantID, email, clientID string, rbac []string) (string, error)
	CreateRefreshToken(ctx context.Context, tenantID, email, clientID string) (string, error)
	ValidateRefreshToken(ctx context.Context, tokenString string) (*tokens.RefreshTokenClaims, error)
	ValidateAccessToken(ctx context.Context, tokenString string) (*tokens.AccessTokenClaims, error)
}

// Authenticated represents the authenticated user's tokens.
type Authenticated struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// AccessTokenClaims represents the claims in an access token.
type AccessTokenClaims struct {
	Roles     []string  `json:"roles"`
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject"`
	Audience  string    `json:"audience"`
	ExpiresAt time.Time `json:"expiresAT"`
	NotBefore time.Time `json:"notBefore"`
	IssuedAt  time.Time `json:"issuedAt"`
	ID        string    `json:"Id,omitempty"`
	ClientID  string    `json:"clientId,omitempty"`
	TenantID  string    `json:"tenantId,omitempty"`
}

// RefreshTokenClaims represents the claims in a refresh token.
type RefreshTokenClaims struct {
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject"`
	Audience  string    `json:"audience"`
	ExpiresAt time.Time `json:"expiresAT"`
	NotBefore time.Time `json:"notBefore"`
	IssuedAt  time.Time `json:"issuedAt"`
	ID        string    `json:"Id,omitempty"`
	ClientID  string    `json:"clientId,omitempty"`
	TenantID  string    `json:"tenantId,omitempty"`
}

// AuthenticationOptions additional configurations
type AuthenticationOptions struct {
	Attempts        int `json:"attempts"`
	LockOutDuration int `json:"lockOutDuration"`
}

// DefaultAuthenticationService is the default implementation of AuthenticationService.
type DefaultAuthenticationService struct {
	accounts                AccountRepository
	tokens                  Tokenizer
	tokenRepository         TokenRepository
	failedAttemptRepository FailedAttemptRepository
	options                 AuthenticationOptions
}

// NewDefaultAuthenticationService creates a new DefaultAuthenticationService.
func NewDefaultAuthenticationService(accounts AccountRepository,
	tokens TokenRepository, tokenizer Tokenizer, failedAttempts FailedAttemptRepository, options AuthenticationOptions) AuthenticationService {
	return &DefaultAuthenticationService{
		accounts:                accounts,
		tokens:                  tokenizer,
		tokenRepository:         tokens,
		failedAttemptRepository: failedAttempts,
		options:                 options,
	}
}

// Authenticate authenticates a user by their email and password.
func (a *DefaultAuthenticationService) Authenticate(ctx context.Context, tenantID, email, clientID, password string) (*Authenticated, error) {
	err := utils.ValidateEmail(email)
	if err != nil {
		return nil, err
	}

	err = utils.ValidatePassword(password)
	if err != nil {
		return nil, err
	}

	// Check if account is locked due to failed attempts
	attempt, err := a.failedAttemptRepository.Get(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	// If lockout period has expired, clear the failed attempts
	if attempt != nil && !attempt.LockedUntil.IsZero() && time.Now().After(attempt.LockedUntil) {
		err = a.failedAttemptRepository.Clear(ctx, tenantID, email)
		if err != nil {
			return nil, err
		}
		attempt = nil
	}

	if attempt != nil && attempt.Count >= a.options.Attempts && time.Now().Before(attempt.LockedUntil) {
		return nil, AccountLockedError{
			Email:       email,
			LockedUntil: attempt.LockedUntil.Format(time.RFC3339),
		}
	}

	account, err := a.accounts.Read(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	err = a.accountHealth(account)
	if err != nil {
		return nil, err
	}

	authenticated, err := a.accounts.PasswordMatches(ctx, tenantID, email, password)
	if err != nil {
		return nil, err
	}

	if !authenticated {
		// Atomically increment failed attempts and lock if threshold reached
		_, err = a.failedAttemptRepository.IncrementAndLockIfNeeded(ctx, tenantID, email,
			a.options.Attempts,
			time.Duration(a.options.LockOutDuration)*time.Second)
		if err != nil {
			return nil, err
		}

		return nil, AuthenticationError{
			Value: email,
		}
	}

	// Clear failed attempts on successful authentication
	err = a.failedAttemptRepository.Clear(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	accessToken, err := a.tokens.CreateAccessToken(ctx, email, tenantID, clientID, account.Roles)
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.tokens.CreateRefreshToken(ctx, tenantID, email, clientID)
	if err != nil {
		return nil, err
	}
	return &Authenticated{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Acknowledge acknowledges the authentication by storing the tokens.
func (a *DefaultAuthenticationService) Acknowledge(ctx context.Context, tenantID string, authenticated Authenticated) error {
	accessClaims, err := a.tokens.ValidateAccessToken(ctx, authenticated.AccessToken)
	if err != nil {
		return err
	}

	if accessClaims.TenantID != tenantID {
		errors.New("token invalid")
	}

	refreshClaims, err := a.tokens.ValidateRefreshToken(ctx, authenticated.RefreshToken)
	if err != nil {
		return err
	}
	if accessClaims.ClientID != refreshClaims.ClientID {
		return errors.New("client IDs do not match")
	}
	err = a.tokenRepository.Create(ctx, tenantID, accessClaims.Subject, accessClaims.ClientID,
		authenticated.AccessToken, authenticated.RefreshToken)
	if err != nil {
		return err
	}
	return nil
}

// ValidateAccessToken validates an access token.
func (a *DefaultAuthenticationService) ValidateAccessToken(ctx context.Context, tenantID, accessToken string) (*AccessTokenClaims, error) {
	token, err := a.tokens.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	if token.TenantID != tenantID {
		return nil, errors.New("token invalid")
	}
	_, err = a.tokenRepository.Read(ctx, token.TenantID, token.Subject, token.ClientID)
	if err != nil {
		return nil, TokenNotAcknowledged{Value: err.Error()}
	}
	return &AccessTokenClaims{
		TenantID:  token.TenantID,
		Roles:     token.Roles,
		Issuer:    token.Issuer,
		Subject:   token.Subject,
		ExpiresAt: token.ExpiresAt.Time,
		NotBefore: token.NotBefore.Time,
		IssuedAt:  token.IssuedAt.Time,
		ID:        token.ID,
		ClientID:  token.ClientID,
	}, nil
}

// ValidateRefreshToken validates a refresh token.
func (a *DefaultAuthenticationService) ValidateRefreshToken(ctx context.Context, tenantID string, refreshToken string) (*RefreshTokenClaims, error) {
	token, err := a.tokens.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	if token.TenantID != tenantID {
		return nil, errors.New("token invalid")
	}

	_, err = a.tokenRepository.Read(ctx, token.TenantID, token.Subject, token.ClientID)
	if err != nil {
		return nil, TokenNotAcknowledged{Value: err.Error()}
	}

	return &RefreshTokenClaims{
		TenantID:  token.TenantID,
		Issuer:    token.Issuer,
		Subject:   token.Subject,
		ExpiresAt: token.ExpiresAt.Time,
		NotBefore: token.NotBefore.Time,
		IssuedAt:  token.IssuedAt.Time,
		ID:        token.ID,
		ClientID:  token.ClientID,
	}, nil
}

// Renew renews the authentication by generating new tokens.
func (a *DefaultAuthenticationService) Renew(ctx context.Context, tenantID, refreshToken string) (*Authenticated, error) {
	token, err := a.tokens.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	if token.TenantID != tenantID {
		return nil, errors.New("token invalid")
	}

	account, err := a.accounts.Read(ctx, token.TenantID, token.Subject)
	if err != nil {
		return nil, err
	}

	accessToken, err := a.tokens.CreateAccessToken(ctx, token.TenantID, token.Subject, token.ClientID, account.Roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err = a.tokens.CreateRefreshToken(ctx, token.TenantID, token.Subject, token.ClientID)
	if err != nil {
		return nil, err
	}

	return &Authenticated{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Revoke revokes the authentication by deleting the tokens.
func (a *DefaultAuthenticationService) Revoke(ctx context.Context, tenantID string, accessToken string) error {
	accessClaims, err := a.tokens.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return err
	}

	if accessClaims.TenantID != tenantID {
		return errors.New("token invalid")
	}

	err = a.tokenRepository.Delete(ctx, accessClaims.TenantID,
		accessClaims.Subject, accessClaims.ClientID)
	if err != nil {
		return err
	}
	return nil
}

func (a *DefaultAuthenticationService) accountHealth(account *accounts.Account) error {
	if account.IsDeleted {
		return accounts.AccountDeletedError{
			Value: account.Email,
		}
	}

	if !account.IsVerified {
		return accounts.AccountNotVerifiedError{
			Value: account.Email,
		}
	}

	if !account.IsEnabled {
		return accounts.AccountDisabledError{
			Value: account.Email,
		}
	}

	return nil
}
