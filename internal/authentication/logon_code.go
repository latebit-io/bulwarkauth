package authentication

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/email"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"golang.org/x/crypto/bcrypt"
)

const (
	codeSize = 6
	expires  = 10 * time.Minute
	charSet  = "1234567890"
)

type LogonCodeService interface {
	Authenticate(ctx context.Context, email, code string) (*Authenticated, error)
	Request(ctx context.Context, email string) error
}

type Encryption interface {
	Encrypt(password string) (string, error)
	Verify(password, verifyPassword string) (bool, error)
}

// EmailService email service contract
type EmailService interface {
	SendMagicLinkEmail(ctx context.Context, email, code string) error
}

type DefaultLogonCodeService struct {
	logonCodeRepository LogonCodeRepository
	accountsRepository  AccountRepository
	encrypt             Encryption
	emailService        email.EmailService
	tokens              tokens.Tokenizer
}

func NewDefaultLogonService(logonRepo LogonCodeRepository, accountsRepository AccountRepository,
	emailService email.EmailService, tokens tokens.Tokenizer, encrypt Encryption) *DefaultLogonCodeService {
	return &DefaultLogonCodeService{
		logonCodeRepository: logonRepo,
		accountsRepository:  accountsRepository,
		encrypt:             encrypt,
		emailService:        emailService,
		tokens:              tokens,
	}
}

func (s *DefaultLogonCodeService) Authenticate(ctx context.Context, email, code string) (*Authenticated, error) {
	compareCode, err := s.logonCodeRepository.Read(ctx, email)
	if err != nil {
		return nil, err
	}

	verified, err := s.encrypt.Verify(compareCode.Code, code)
	if err != nil {
		return nil, err
	}

	if verified {
		account, err := s.accountsRepository.Read(ctx, email)
		if err != nil {
			return nil, err
		}

		accessToken, err := s.tokens.CreateAccessToken(ctx, email, account.Roles)
		if err != nil {
			return nil, err
		}
		refreshToken, err := s.tokens.CreateRefreshToken(ctx, email)
		if err != nil {
			return nil, err
		}
		err = s.logonCodeRepository.Delete(ctx, email, compareCode.Code)
		if err != nil {
			log.Println(err)
		}
		return &Authenticated{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil
	}

	return nil, AuthenticationError{
		Value: email,
	}
}

func (s *DefaultLogonCodeService) Request(ctx context.Context, email string) error {
	_, err := s.accountsRepository.Read(ctx, email)
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

	err = s.logonCodeRepository.Create(ctx, email, string(hashedCode), time.Now().Add(expires))
	if err != nil {
		return err
	}

	err = s.emailService.SendMagicLinkEmail(ctx, email, code)
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
