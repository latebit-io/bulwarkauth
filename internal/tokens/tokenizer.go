package tokens

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Tokenizer interface {
	CreateAccessToken(ctx context.Context, email string, rbac []string) (string, error)
	CreateRefreshToken(ctx context.Context, email string) (string, error)
	ValidateRefreshToken(ctx context.Context, email, token string) (*RefreshTokenClaims, error)
	ValidateAccessToken(ctx context.Context, email, tokenString string) (*AccessTokenClaims, error)
}

type DefaultTokenizer struct {
	Name     string
	Issuer   string
	Audience string

	keys                 map[string]SigningKey
	signingKeyService    SigningKeyService
	refreshTokenExpInSec int
	accessTokenExpInSec  int
}

type AccessTokenClaims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	jwt.RegisteredClaims
}

func NewDefaultTokenizer(name, issuer, audience string, refreshTokenExpInSec int, accessTokenExpInSec int, service SigningKeyService) *DefaultTokenizer {
	keys, err := service.GetAllKeys(context.Background())
	keyMap := make(map[string]SigningKey)
	for _, k := range keys {
		keyMap[k.KeyId] = k
	}
	if err != nil {
		panic(err)
	}
	return &DefaultTokenizer{
		Name:     name,
		Issuer:   issuer,
		Audience: audience,

		keys:                 keyMap,
		signingKeyService:    service,
		accessTokenExpInSec:  accessTokenExpInSec,
		refreshTokenExpInSec: refreshTokenExpInSec,
	}
}

func (d DefaultTokenizer) CreateAccessToken(ctx context.Context, email string, rbac []string) (string, error) {
	key, err := d.signingKeyService.LatestKey(ctx)
	if err != nil {
		return "", err
	}

	id := uuid.New()

	claims := AccessTokenClaims{
		Roles: rbac,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(d.accessTokenExpInSec))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    d.Issuer,
			Subject:   email,
			Audience:  []string{d.Audience},
		},
	}

	j := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	j.Header["use"] = "access"
	j.Header["kid"] = key.KeyId
	privateKey, err := d.loadPrivateKey(key.PrivateKey)
	if err != nil {
		return "", err
	}

	token, err := j.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (d DefaultTokenizer) CreateRefreshToken(ctx context.Context, email string) (string, error) {
	key, err := d.signingKeyService.LatestKey(ctx)
	if err != nil {
		return "", err
	}

	id := uuid.New()

	claims := RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(d.refreshTokenExpInSec))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    d.Issuer,
			Subject:   email,
			Audience:  []string{d.Audience},
		},
	}

	j := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	j.Header["use"] = "refresh"
	j.Header["kid"] = key.KeyId
	privateKey, err := d.loadPrivateKey(key.PrivateKey)
	if err != nil {
		return "", err
	}

	token, err := j.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (d DefaultTokenizer) ValidateRefreshToken(ctx context.Context, email, tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("signing method not supported")
		}
		tokenEmail, err := token.Claims.GetSubject()

		if err != nil {
			return nil, err
		}

		if tokenEmail != email {
			return nil, fmt.Errorf("invalid token")
		}

		kid := fmt.Sprintf("%v", token.Header["kid"])
		key := d.keys[kid]
		return d.loadPublicKey(key.PublicKey)
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func (d DefaultTokenizer) ValidateAccessToken(ctx context.Context, email, tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("signing method not supported")
		}

		tokenEmail, err := token.Claims.GetSubject()

		if err != nil {
			return nil, err
		}

		if tokenEmail != email {
			return nil, fmt.Errorf("invalid token")
		}

		kid := fmt.Sprintf("%v", token.Header["kid"])
		key := d.keys[kid]
		return d.loadPublicKey(key.PublicKey)
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AccessTokenClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func (d DefaultTokenizer) loadPrivateKey(key string) (*rsa.PrivateKey, error) {
	keyBytes := []byte(key)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func (d DefaultTokenizer) loadPublicKey(key string) (*rsa.PublicKey, error) {
	keyBytes := []byte(key)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return publicKey.(*rsa.PublicKey), nil
}
