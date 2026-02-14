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
	return fmt.Sprintf("api key invalid: %s", e.Value)
}

type ApiKeyDisabledError struct {
	Value string
}

func (e ApiKeyDisabledError) Error() string {
	return fmt.Sprintf("api key is disabled: %s", e.Value)
}

type ApiKeyExpiredError struct {
	Value string
}

func (e ApiKeyExpiredError) Error() string {
	return fmt.Sprintf("api key is expired: %s", e.Value)
}
