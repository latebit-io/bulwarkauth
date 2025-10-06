package domain

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

// ClientCredentialsRepository interface for storing and retrieving client credentials
type ClientCredentialsRepository interface {
	GetCredentialsByClientID(clientID string) (*DomainCredentials, error)
}

// verifyClientCredentials verifies the client ID and signature of the request
func verifyClientCredentials(clientID string, signature string, r *http.Request, repo ClientCredentialsRepository) bool {
	// Get client credentials from repository
	creds, err := repo.GetCredentialsByClientID(clientID)
	if err != nil {
		return false
	}

	// Recreate the signature using the request data
	payload, err := getRequestPayload(r)
	if err != nil {
		return false
	}

	// Create timestamp validation
	timestamp := r.Header.Get("X-Timestamp")
	if !isValidTimestamp(timestamp) {
		return false
	}

	// Combine payload with timestamp for signature
	signatureBase := append(payload, []byte(timestamp)...)

	// Calculate expected signature
	expectedSignature := calculateSignature(creds.ClientKey, signatureBase)

	// Compare signatures using constant-time comparison
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// getRequestPayload gets the request body for signature verification
func getRequestPayload(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return []byte{}, nil
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Restore the body for subsequent reads
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return body, nil
}

// calculateSignature creates HMAC signature using client key
func calculateSignature(clientKey string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(clientKey))
	h.Write(payload)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// isValidTimestamp verifies the request timestamp is within acceptable range
func isValidTimestamp(timestamp string) bool {
	// Parse the timestamp
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false
	}

	// Check if timestamp is within 5 minutes of current time
	// This helps prevent replay attacks
	now := time.Now()
	diff := now.Sub(ts)
	return diff.Minutes() <= 5
}

// Example usage in middleware
func AuthenticateRequest(repo ClientCredentialsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID := r.Header.Get("X-Client-ID")
			signature := r.Header.Get("X-Signature")

			if clientID == "" || signature == "" {
				http.Error(w, "Missing authentication headers", http.StatusUnauthorized)
				return
			}

			if !verifyClientCredentials(clientID, signature, r, repo) {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
