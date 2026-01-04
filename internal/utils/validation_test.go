package utils

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with dots",
			email:   "first.last@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with numbers",
			email:   "user123@example456.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with dash in domain",
			email:   "user@my-domain.com",
			wantErr: false,
		},
		{
			name:    "valid email with underscore",
			email:   "user_name@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with percent",
			email:   "user%test@example.com",
			wantErr: false,
		},
		{
			name:    "invalid email - no @",
			email:   "userexample.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - no domain",
			email:   "user@",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - no local part",
			email:   "@example.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - no TLD",
			email:   "user@example",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - spaces",
			email:   "user @example.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - multiple @",
			email:   "user@@example.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - special chars",
			email:   "user#test@example.com",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - too long",
			email:   strings.Repeat("a", 245) + "@example.com", // 245 + 12 = 257 chars
			wantErr: true,
			errMsg:  "email too long",
		},
		{
			name:    "valid email - exactly 254 chars",
			email:   strings.Repeat("a", 242) + "@example.com", // 242 + 12 = 254 chars
			wantErr: false,
		},
		{
			name:    "invalid email - empty",
			email:   "",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "invalid email - TLD too short",
			email:   "user@example.c",
			wantErr: true,
			errMsg:  "invalid email format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateEmail() expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidateEmail() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEmail() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password - 8 chars",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "valid password - 16 chars",
			password: "1234567890abcdef",
			wantErr:  false,
		},
		{
			name:     "valid password - 128 chars",
			password: strings.Repeat("a", 128),
			wantErr:  false,
		},
		{
			name:     "valid password - special chars",
			password: "P@ssw0rd!#$%",
			wantErr:  false,
		},
		{
			name:     "valid password - unicode",
			password: "pässwörd123",
			wantErr:  false,
		},
		{
			name:     "valid password - spaces",
			password: "my password 123",
			wantErr:  false,
		},
		{
			name:     "invalid password - too short (7 chars)",
			password: "1234567",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters",
		},
		{
			name:     "invalid password - too short (empty)",
			password: "",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters",
		},
		{
			name:     "invalid password - too long (129 chars)",
			password: strings.Repeat("a", 129),
			wantErr:  true,
			errMsg:   "password too long",
		},
		{
			name:     "invalid password - way too long",
			password: strings.Repeat("a", 1000),
			wantErr:  true,
			errMsg:   "password too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePassword() expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidatePassword() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePassword() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		size    int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid code - 6 digits",
			code:    "123456",
			size:    6,
			wantErr: false,
		},
		{
			name:    "valid code - 4 digits",
			code:    "1234",
			size:    4,
			wantErr: false,
		},
		{
			name:    "valid code - 8 chars",
			code:    "abcd1234",
			size:    8,
			wantErr: false,
		},
		{
			name:    "valid code - single char",
			code:    "a",
			size:    1,
			wantErr: false,
		},
		{
			name:    "valid code - empty with size 0",
			code:    "",
			size:    0,
			wantErr: false,
		},
		{
			name:    "invalid code - too short",
			code:    "12345",
			size:    6,
			wantErr: true,
			errMsg:  "invalid code",
		},
		{
			name:    "invalid code - too long",
			code:    "1234567",
			size:    6,
			wantErr: true,
			errMsg:  "invalid code",
		},
		{
			name:    "invalid code - empty when expecting 6",
			code:    "",
			size:    6,
			wantErr: true,
			errMsg:  "invalid code",
		},
		{
			name:    "invalid code - wrong size",
			code:    "abc",
			size:    5,
			wantErr: true,
			errMsg:  "invalid code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCode(tt.code, tt.size)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCode() expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidateCode() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCode() unexpected error = %v", err)
				}
			}
		})
	}
}
