package encryption

import (
	"golang.org/x/crypto/bcrypt"
)

type Encryption interface {
	Encrypt(password string) (string, error)
	Verify(password, verifyPassword string) (bool, error)
}

type DefaultEncryption struct {
}

func NewDefaultEncryption() *DefaultEncryption {
	return &DefaultEncryption{}
}

func (d DefaultEncryption) Encrypt(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
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
