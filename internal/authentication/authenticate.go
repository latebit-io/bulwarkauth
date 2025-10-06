package authentication

import (
	"context"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
)

// AuthenticationService defines the interface for authentication services.
type AuthenticationService interface {
	Authenticate(ctx context.Context, email string, password string) (*Authenticated, error)
	Acknowledge(ctx context.Context, Authenticate Authenticated, email, clientId string) error
	ValidateAccessToken(ctx context.Context, accessToken, email string) (*AccessTokenClaims, error)
	ValidateRefreshToken(ctx context.Context, refreshToken, email string) (*RefreshTokenClaims, error)
	Renew(ctx context.Context, refreshToken, clientId string) (*Authenticated, error)
	Revoke(ctx context.Context, clientId, email string, accessToken string) error
}

type AccountRepository interface {
	Read(ctx context.Context, email string) (*accounts.Account, error)
	PasswordMatches(ctx context.Context, email, password string) (bool, error)
}

type Tokenizer interface {
	CreateAccessToken(ctx context.Context, email string, rbac []string) (string, error)
	CreateRefreshToken(ctx context.Context, email string) (string, error)
	ValidateRefreshToken(ctx context.Context, email, tokenString string) (*tokens.RefreshTokenClaims, error)
	ValidateAccessToken(ctx context.Context, email, tokenString string) (*tokens.AccessTokenClaims, error)
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
}

// DefaultAuthenticationService is the default implementation of AuthenticationService.
type DefaultAuthenticationService struct {
	accounts        AccountRepository
	tokens          Tokenizer
	tokenRepository TokenRepository
}

// NewDefaultAuthenticationService creates a new DefaultAuthenticationService.
func NewDefaultAuthenticationService(accounts AccountRepository, tokens TokenRepository, tokenizer Tokenizer) *DefaultAuthenticationService {
	return &DefaultAuthenticationService{
		accounts:        accounts,
		tokens:          tokenizer,
		tokenRepository: tokens,
	}
}

// Authenticate authenticates a user by their email and password.
func (a *DefaultAuthenticationService) Authenticate(ctx context.Context, email string, password string) (*Authenticated, error) {
	account, err := a.accounts.Read(ctx, email)
	if err != nil {
		return nil, err
	}

	err = a.accountHealth(account)
	if err != nil {
		return nil, err
	}

	authenticated, err := a.accounts.PasswordMatches(ctx, email, password)
	if err != nil {
		return nil, err
	}

	if !authenticated {
		return nil, AuthenticationError{
			Value: email,
		}
	}

	accessToken, err := a.tokens.CreateAccessToken(ctx, email, nil)
	if err != nil {
		return nil, err
	}
	refreshToken, err := a.tokens.CreateRefreshToken(ctx, email)
	if err != nil {
		return nil, err
	}
	return &Authenticated{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Acknowledge acknowledges the authentication by storing the tokens.
func (a *DefaultAuthenticationService) Acknowledge(ctx context.Context, authenticated Authenticated, email, clientId string) error {
	err := a.tokenRepository.Create(ctx, email, clientId, authenticated.AccessToken, authenticated.RefreshToken)
	//TODO: on ack have the option to set cookie for SPA
	if err != nil {
		return err
	}
	return nil
}

// ValidateAccessToken validates an access token.
func (a *DefaultAuthenticationService) ValidateAccessToken(ctx context.Context, accessToken string, email string) (*AccessTokenClaims, error) {
	token, err := a.tokens.ValidateAccessToken(ctx, email, accessToken)
	if err != nil {
		return nil, err
	}
	return &AccessTokenClaims{
		Roles:     token.Roles,
		Issuer:    token.Issuer,
		Subject:   token.Subject,
		ExpiresAt: token.ExpiresAt.Time,
		NotBefore: token.NotBefore.Time,
		IssuedAt:  token.IssuedAt.Time,
		ID:        token.ID,
	}, nil
}

// ValidateRefreshToken validates a refresh token.
func (a *DefaultAuthenticationService) ValidateRefreshToken(ctx context.Context, refreshToken string, email string) (*RefreshTokenClaims, error) {
	token, err := a.tokens.ValidateRefreshToken(ctx, email, refreshToken)
	if err != nil {
		return nil, err
	}
	return &RefreshTokenClaims{
		Issuer:    token.Issuer,
		Subject:   token.Subject,
		ExpiresAt: token.ExpiresAt.Time,
		NotBefore: token.NotBefore.Time,
		IssuedAt:  token.IssuedAt.Time,
		ID:        token.ID,
	}, nil
}

// Renew renews the authentication by generating new tokens.
func (a *DefaultAuthenticationService) Renew(ctx context.Context, refreshToken, email string) (*Authenticated, error) {
	token, err := a.tokens.ValidateRefreshToken(ctx, email, refreshToken)
	if err != nil {
		return nil, err
	}

	account, err := a.accounts.Read(ctx, token.Subject)
	if err != nil {
		return nil, err
	}

	accessToken, err := a.tokens.CreateAccessToken(ctx, token.Subject, account.Roles)
	if err != nil {
		return nil, err
	}

	refreshToken, err = a.tokens.CreateRefreshToken(ctx, token.Subject)
	if err != nil {
		return nil, err
	}

	return &Authenticated{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Revoke revokes the authentication by deleting the tokens.
func (a *DefaultAuthenticationService) Revoke(ctx context.Context, clientId, email string, accessToken string) error {
	err := a.tokenRepository.Delete(ctx, email, clientId)
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
