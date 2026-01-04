package utils

import (
	"errors"
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) error {
	if len(email) > 254 {
		return errors.New("email too long")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return errors.New("password too long")
	}
	return nil
}

func ValidateCode(code string, size int) error {
	if len(code) != size {
		return errors.New("invalid code")
	}
	return nil
}
