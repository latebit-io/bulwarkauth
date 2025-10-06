package tokens

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SigningKey struct {
	KeyId      string    `bson:"key_id"`
	Format     string    `bson:"format"`
	Algorithm  string    `bson:"algorithm"`
	PrivateKey string    `bson:"private_key"`
	PublicKey  string    `bson:"public_key"`
	Created    time.Time `bson:"created"`
}

func NewSigningKey(byteSize int) (*SigningKey, error) {
	bits := byteSize * 8
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	publicKey := &privateKey.PublicKey
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	keyId := uuid.New()
	return &SigningKey{
		KeyId:      keyId.String(),
		Algorithm:  fmt.Sprintf("RS%d", byteSize),
		PrivateKey: pemBlockToString(privateKeyPEM),
		PublicKey:  pemBlockToString(publicKeyPEM),
		Format:     "PKCS#1",
		Created:    time.Now(),
	}, nil
}

func pemBlockToString(block *pem.Block) string {
	return string(pem.EncodeToMemory(block))
}
