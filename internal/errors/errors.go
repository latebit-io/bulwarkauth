package errors

import "fmt"

type DuplicateError struct {
	Value string `json:"value"`
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("duplicate entry: '%s' already exists", e.Value)
}

type NotFoundError struct {
	Value string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Value)
}
