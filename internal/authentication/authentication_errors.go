package authentication

import "fmt"

type AuthenticationError struct {
	Value string
}

func (e AuthenticationError) Error() string {
	return fmt.Sprintf("cannot authenticate account: %s", e.Value)
}

type TokenNotAcknowledged struct {
	Value string
}

func (e TokenNotAcknowledged) Error() string {
	return fmt.Sprintf("token not acknowledged: %s", e.Value)
}

type AccountLockedError struct {
	Email       string
	LockedUntil string
}

func (e AccountLockedError) Error() string {
	return fmt.Sprintf("account is temporarily locked due, too many attempts")
}
