package accounts

import "fmt"

type AccountDuplicateError struct {
	Value string `json:"value"`
}

func (e AccountDuplicateError) Error() string {
	return fmt.Sprintf("duplicate account: '%s' already exists", e.Value)
}

type AccountNotFoundError struct {
	Value string `json:"value"`
}

func (e AccountNotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Value)
}

type VerificationError struct {
	Value string `json:"value"`
}

func (e VerificationError) Error() string { return fmt.Sprintf("verification error: %s", e.Value) }

type AccountDeletedError struct {
	Value string `json:"value"`
}

func (e AccountDeletedError) Error() string {
	return fmt.Sprintf("account: '%s' is deleted", e.Value)
}

type AccountNotVerifiedError struct {
	Value string `json:"value"`
}

func (e AccountNotVerifiedError) Error() string {
	return fmt.Sprintf("account: '%s' is not verified", e.Value)
}

type AccountDisabledError struct {
	Value string `json:"value"`
}

func (e AccountDisabledError) Error() string {
	return fmt.Sprintf("account: '%s' is disabled", e.Value)
}
