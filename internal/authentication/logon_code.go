package authentication

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/email"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	codeSize = 6
	expires  = 10 * time.Minute
	charSet  = "1234567890"
)

type LogonCodeService interface {
	Authenticate(ctx context.Context, tenantID, email, clientID, code string) (*Authenticated, error)
	Request(ctx context.Context, tenantID, email string) error
}

type Encryption interface {
	Encrypt(password string) (string, error)
	Verify(password, verifyPassword string) (bool, error)
}

// EmailService email service contract
type EmailService interface {
	SendMagicLinkEmail(ctx context.Context, tenantID, email, code string) error
}

type DefaultLogonCodeService struct {
	logonCodeRepository     LogonCodeRepository
	accountsRepository      AccountRepository
	encrypt                 Encryption
	emailService            email.EmailService
	tokens                  tokens.Tokenizer
	failedAttemptRepository FailedAttemptRepository
	options                 AuthenticationOptions
}

func NewDefaultLogonService(logonRepo LogonCodeRepository, accountsRepository AccountRepository,
	emailService email.EmailService, tokens tokens.Tokenizer, encrypt Encryption, failedAttempts FailedAttemptRepository, options AuthenticationOptions) LogonCodeService {
	return &DefaultLogonCodeService{
		logonCodeRepository:     logonRepo,
		accountsRepository:      accountsRepository,
		encrypt:                 encrypt,
		emailService:            emailService,
		tokens:                  tokens,
		failedAttemptRepository: failedAttempts,
		options:                 options,
	}
}

func (s *DefaultLogonCodeService) Authenticate(ctx context.Context, tenantID, email, clientID, code string) (*Authenticated, error) {
	err := utils.ValidateEmail(email)
	if err != nil {
		return nil, err
	}

	err = utils.ValidateCode(code, codeSize)
	if err != nil {
		return nil, err
	}

	// Check if account is locked due to failed attempts
	attempt, err := s.failedAttemptRepository.Get(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	// If lockout period has expired, clear the failed attempts
	if attempt != nil && !attempt.LockedUntil.IsZero() && time.Now().After(attempt.LockedUntil) {
		err = s.failedAttemptRepository.Clear(ctx, tenantID, email)
		if err != nil {
			return nil, err
		}
		attempt = nil
	}

	if attempt != nil && attempt.Count >= s.options.Attempts && time.Now().Before(attempt.LockedUntil) {
		return nil, AccountLockedError{
			Email:       email,
			LockedUntil: attempt.LockedUntil.Format(time.RFC3339),
		}
	}

	compareCode, err := s.logonCodeRepository.Read(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	verified, err := s.encrypt.Verify(compareCode.Code, code)
	if err != nil {
		return nil, err
	}

	if verified {
		// Clear failed attempts on successful authentication
		err = s.failedAttemptRepository.Clear(ctx, tenantID, email)
		if err != nil {
			return nil, err
		}

		account, err := s.accountsRepository.Read(ctx, tenantID, email)
		if err != nil {
			return nil, err
		}

		accessToken, err := s.tokens.CreateAccessToken(ctx, tenantID, email, clientID, account.Roles)
		if err != nil {
			return nil, err
		}
		refreshToken, err := s.tokens.CreateRefreshToken(ctx, tenantID, email, clientID)
		if err != nil {
			return nil, err
		}
		err = s.logonCodeRepository.Delete(ctx, tenantID, email, compareCode.Code)
		if err != nil {
			log.Println(err)
		}
		return &Authenticated{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil
	}

	// Atomically increment failed attempts and lock if threshold reached
	_, err = s.failedAttemptRepository.IncrementAndLockIfNeeded(ctx, tenantID, email,
		s.options.Attempts,
		time.Duration(s.options.LockOutDuration)*time.Second)
	if err != nil {
		return nil, err
	}

	return nil, AuthenticationError{
		Value: email,
	}
}

func (s *DefaultLogonCodeService) Request(ctx context.Context, tenantID, email string) error {
	_, err := s.accountsRepository.Read(ctx, tenantID, email)
	if err != nil {
		return err
	}

	code, err := GetUniqueKey(codeSize)
	if err != nil {
		return err
	}

	hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	err = s.logonCodeRepository.Create(ctx, tenantID, email, string(hashedCode), time.Now().Add(expires))
	if err != nil {
		return err
	}

	err = s.emailService.SendMagicLinkEmail(ctx, tenantID, email, code)
	if err != nil {
		return err
	}

	return nil
}

func GetUniqueKey(size int) (string, error) {
	b := make([]byte, size)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			return "", err
		}
		b[i] = charSet[num.Int64()]
	}
	return string(b), nil
}
