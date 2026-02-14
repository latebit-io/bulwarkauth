package encryption

import (
	"errors"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Encryption interface {
	Encrypt(password string) (string, error)
	Verify(password, verifyPassword string) (bool, error)
}

type DefaultEncryption struct {
	cost int
}

// NewDefaultEncryption creates a new instance of DefaultEncryption with the specified cost.
// Default cost is 12, if set to lower than 12, it will be set to 12.
func NewDefaultEncryption(cost int) *DefaultEncryption {
	if cost < 12 {
		cost = 12
	}
	return &DefaultEncryption{
		cost: cost,
	}
}

func (d DefaultEncryption) Encrypt(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), d.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (d DefaultEncryption) Verify(password, verifyPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(password), []byte(verifyPassword))
	if err != nil {
		// If the password doesn't match, return false without error
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		// For other bcrypt errors (invalid hash format, etc.), return the error
		return false, err
	}
	return true, nil
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateRandomString generates a random string of the specified length
func GenerateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
