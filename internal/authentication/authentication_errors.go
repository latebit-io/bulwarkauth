package authentication

import "fmt"

type AuthenticationError struct {
	Value string `json:"value"`
}

func (e AuthenticationError) Error() string {
	return fmt.Sprintf("cannot authenticate account: %s", e.Value)
}
