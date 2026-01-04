package encryption

import (
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
// default cost is 12, if set to lower than 12, it will be set to 12.
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
		return false, err
	}
	return true, nil
}
