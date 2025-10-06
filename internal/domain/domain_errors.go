package domain

import "fmt"

type DuplicateVerificationError struct {
	Value string `json:"value"`
}

func (e DuplicateVerificationError) Error() string {
	return fmt.Sprintf("duplicate domain: %s already exists", e.Value)
}

type VerificationError struct {
	Value string `json:"value"`
}

func (e VerificationError) Error() string {
	return fmt.Sprintf("domain verification error: %s", e.Value)
}
