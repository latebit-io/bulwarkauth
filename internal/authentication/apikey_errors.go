package authentication

import "fmt"

type ApiKeyNotFoundError struct {
	Value string
}

func (e ApiKeyNotFoundError) Error() string {
	return fmt.Sprintf("api key not found: %s", e.Value)
}

type ApiKeyInvalidError struct {
	Value string
}

func (e ApiKeyInvalidError) Error() string {
	return fmt.Sprintf("api key not valid: %s", e.Value)
}

type ApiKeyDisabledError struct {
	Value string
}

func (e ApiKeyDisabledError) Error() string {
	return fmt.Sprintf("api key disabled: %s", e.Value)
}

type ApiKeyExpiredError struct {
	Value string
}

func (e ApiKeyExpiredError) Error() string {
	return fmt.Sprintf("api key expired: %s", e.Value)
}
