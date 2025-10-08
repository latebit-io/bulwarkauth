package accounts

import (
	"context"
	"errors"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"go.mongodb.org/mongo-driver/mongo"
)

// AccountService contract for all account related actions
type AccountService interface {
	Create(ctx context.Context, email string, password string) error
	Verify(ctx context.Context, email string, verificationCode string) error
	Resend(ctx context.Context, email string) error
	UpdateEmail(ctx context.Context, email string, accessToken string) error
	Delete(ctx context.Context, email string, accessToken string) error
	UpdatePassword(ctx context.Context, email, newPassword, accessToken string) error
	Forgot(ctx context.Context, email string) error
	ForgotPassword(ctx context.Context, email, newPassword, forgotToken string) error
}

type EmailService interface {
	SendVerificationEmail(ctx context.Context, email, verificationToken string) error
	SendForgotPasswordEmail(ctx context.Context, email, forgotToken string) error
	SendMagicLinkEmail(ctx context.Context, email, code string) error
}

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx mongo.SessionContext) error) error
}

type Tokenizer interface {
	ValidateAccessToken(ctx context.Context, email, tokenString string) (*tokens.AccessTokenClaims, error)
}

type Account struct {
	Email             string           `bson:"email"`
	IsVerified        bool             `bson:"isVerified"`
	VerificationToken string           `bson:"verificationToken"`
	IsEnabled         bool             `bson:"isEnabled"`
	IsDeleted         bool             `bson:"isDeleted"`
	SocialProviders   []SocialProvider `bson:"socialProviders"`
	Roles             []string         `bson:"roles"`
	Created           time.Time        `bson:"created"`
	Modified          time.Time        `bson:"modified"`
}

type SocialProvider struct {
	Name     string `bson:"name" json:"name"`
	SocialId string `bson:"socialId" json:"socialId"`
}

type DefaultAccountService struct {
	accountRepository AccountRepository
	forgotRepository  ForgotRepository
	tokenizer         Tokenizer
	emailService      EmailService
	txManager         TxManager
}

func NewDefaultAccountService(accountRepository AccountRepository, forgotRepository ForgotRepository,
	tokenizer Tokenizer, emailService EmailService, txManager TxManager) AccountService {
	return DefaultAccountService{
		accountRepository: accountRepository,
		tokenizer:         tokenizer,
		forgotRepository:  forgotRepository,
		emailService:      emailService,
		txManager:         txManager,
	}
}

// Resend will send the verification email if the account has not yet been verified
func (a DefaultAccountService) Resend(ctx context.Context, email string) error {
	account, err := a.accountRepository.Read(ctx, email)
	if err != nil {
		return err
	}
	if !account.IsVerified {
		return a.emailService.SendVerificationEmail(ctx, email, account.VerificationToken)
	}

	return VerificationError{
		Value: "no token",
	}
}

// ForgotPassword will reset a users password if supplied with a valid token
func (a DefaultAccountService) ForgotPassword(ctx context.Context, email, newPassword, forgotToken string) error {
	forgot, err := a.forgotRepository.Read(ctx, email)
	if err != nil {
		return err
	}
	if forgot.Token != forgotToken {
		return errors.New("cannot change password")
	}

	return a.txManager.WithTransaction(ctx, func(txCtx mongo.SessionContext) error {
		err = a.accountRepository.UpdatePassword(ctx, email, newPassword)
		if err != nil {
			return err
		}
		err = a.forgotRepository.Delete(ctx, email)
		if err != nil {
			return err
		}
		return nil
	})
}

// Forgot will send a forgot email using the forgot template te the user with a link to reset their password
// the service can be configured with a endpoint that will call the forgot password bulwark api
func (a DefaultAccountService) Forgot(ctx context.Context, email string) error {
	err := a.forgotRepository.Create(ctx, email)
	if err != nil {
		return err
	}

	forgot, err := a.forgotRepository.Read(ctx, email)
	if err != nil {
		return err
	}

	err = a.emailService.SendForgotPasswordEmail(ctx, email, forgot.Token)
	if err != nil {
		return err
	}

	return nil
}

// Create will create a new user if the email is available
func (a DefaultAccountService) Create(ctx context.Context, email string, password string) error {
	err := a.accountRepository.Create(ctx, email, password)
	if err != nil {
		return err
	}
	account, err := a.accountRepository.Read(ctx, email)
	if err != nil {
		return err
	}
	return a.emailService.SendVerificationEmail(ctx, email, account.VerificationToken)
}

// Verify when an account ot email is changed an account will need to be verified
func (a DefaultAccountService) Verify(ctx context.Context, email string, verificationCode string) error {
	account, err := a.accountRepository.Read(ctx, email)
	if err != nil {
		return err
	}

	if account.VerificationToken != verificationCode {
		return VerificationError{
			Value: "cannot verify account",
		}
	}

	return a.accountRepository.Verify(ctx, email)
}

// UpdateEmail updates an accounts email must supply a valid accessToken
func (a DefaultAccountService) UpdateEmail(ctx context.Context, email string, accessToken string) error {
	token, err := a.tokenizer.ValidateAccessToken(ctx, email, accessToken)
	if err != nil {
		return err
	}
	newVerification, err := a.accountRepository.UpdateEmail(ctx, token.Subject, email)
	if err != nil {
		return err
	}

	err = a.emailService.SendVerificationEmail(ctx, email, newVerification.Token)
	if err != nil {
		return err
	}

	return nil
}

func (a DefaultAccountService) UpdatePassword(ctx context.Context, email, newPassword, accessToken string) error {
	_, err := a.tokenizer.ValidateAccessToken(ctx, email, accessToken)
	if err != nil {
		return err
	}

	if err = a.accountRepository.UpdatePassword(ctx, email, newPassword); err != nil {
		return err
	}

	return nil
}

func (a DefaultAccountService) Delete(ctx context.Context, email string, accessToken string) error {
	_, err := a.tokenizer.ValidateAccessToken(ctx, email, accessToken)
	if err != nil {
		return err
	}

	err = a.accountRepository.Delete(ctx, email)
	if err != nil {
		return err
	}

	return nil
}
