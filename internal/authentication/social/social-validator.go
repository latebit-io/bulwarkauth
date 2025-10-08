package social

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/authentication"
	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
)

type Validator interface {
	Name() string
	ValidateToken(ctx context.Context, ID string) (*Social, error)
}

type Social struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Provider string `json:"provider"`
}

type GoogleValidator struct {
	name     string
	verifier *oidc.IDTokenVerifier
}

type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"` // Google user ID
}

func NewGoogleValidator(clientID string) (*GoogleValidator, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID, // Your Google OAuth client ID
	})

	return &GoogleValidator{name: "google", verifier: verifier}, nil
}

func (gv *GoogleValidator) Name() string {
	return "google"
}

func (gv *GoogleValidator) ValidateToken(ctx context.Context, ID string) (*Social, error) {
	token, err := gv.verifier.Verify(ctx, ID)
	if err != nil {
		return nil, err
	}
	var claims GoogleClaims
	if err := token.Claims(&claims); err != nil {
		return nil, err
	}
	return &Social{
		ID:       claims.Sub,
		Email:    claims.Email,
		Provider: gv.Name(),
	}, nil
}

type SocialService interface {
	AddValidator(validator Validator)
	Authenticate(context context.Context, idToken, provider string) (*authentication.Authenticated, error)
}

type DefaultSocialService struct {
	validators     map[string]Validator
	accountRepo    accounts.AccountRepository
	accountService accounts.AccountService
	encrypt        encryption.Encryption
	token          tokens.Tokenizer
}

func NewDefaultSocialService(accountRepo accounts.AccountRepository,
	accountService accounts.AccountService, encryption encryption.Encryption, token tokens.Tokenizer) *DefaultSocialService {
	return &DefaultSocialService{
		validators:     make(map[string]Validator),
		accountRepo:    accountRepo,
		accountService: accountService,
		encrypt:        encryption,
		token:          token,
	}
}

func (s *DefaultSocialService) AddValidator(validator Validator) {
	s.validators[validator.Name()] = validator
}

func (s *DefaultSocialService) Authenticate(ctx context.Context, idToken, provider string) (*authentication.Authenticated, error) {
	validator, ok := s.validators[provider]
	if !ok {
		return nil, fmt.Errorf("no validator found for provider %s", provider)
	}
	social, err := validator.ValidateToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	if social.Email == "" {
		return nil, fmt.Errorf("no email found in ID token %s", provider)
	}

	account, err := s.accountRepo.Read(ctx, social.Email)
	var notFound accounts.AccountNotFoundError
	if errors.As(err, &notFound) {
		randomPassword := uuid.New().String()
		err = s.accountService.Create(ctx, social.Email, randomPassword)
		if err != nil {
			return nil, err
		}
		return nil, accounts.AccountNotVerifiedError{Value: social.Email}
	} else if err != nil {
		return nil, err
	}

	err = s.accountRepo.LinkSocial(ctx, social.Email, accounts.SocialProvider{
		Name:     social.Provider,
		SocialId: social.ID,
	})

	if err != nil {
		return nil, err
	}

	accessToken, err := s.token.CreateAccessToken(ctx, account.Email, account.Roles)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.token.CreateRefreshToken(ctx, account.Email)
	if err != nil {
		return nil, err
	}

	return &authentication.Authenticated{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
