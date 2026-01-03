package tokens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSigningKey(t *testing.T) {
	newSigningKey, err := NewSigningKey(256)
	assert.Equal(t, nil, err)
	assert.NotEmpty(t, newSigningKey.PrivateKey)
	assert.NotEmpty(t, newSigningKey.PublicKey)
}
