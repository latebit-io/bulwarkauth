package tokens

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSigningKey(t *testing.T) {
	newSigningKey, err := NewSigningKey(256)
	assert.Equal(t, nil, err)
	fmt.Println(newSigningKey.PrivateKey)
	fmt.Println(newSigningKey.PublicKey)
}
