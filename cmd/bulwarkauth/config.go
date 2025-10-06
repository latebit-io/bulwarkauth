package main

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	AccessTokenExpireInSeconds  int
	AllowedOrigins              []string
	ApiKeyEnabled               bool
	CompanyID                   string
	CORSEnabled                 bool
	DbConnection                string
	DbNameSeed                  string
	Domain                      string
	DomainVerify                bool
	EmailAuth                   bool
	EmailFromAddress            string
	EmailSmtpHost               string
	EmailSmtpPass               string
	EmailSmtpPort               string
	EmailSmtpSecure             bool
	EmailSmtpUser               string
	EmailTemplateDir            string
	EmailTemplatesDir           string
	EnableSmtp                  bool
	ForgotPasswordUrl           string
	GithubAppName               string
	GoogleClientId              string
	MagicCodeExpireInMinutes    int
	MagicUrl                    string
	MicrosoftClientId           string
	MicrosoftTenantId           string
	Port                        int
	RefreshTokenExpireInSeconds int
	VerificationUrl             string
	WebsiteName                 string
	TestMode                    bool
}

func NewAppConfig() (*AppConfig, error) {
	config := &AppConfig{}

	config.Port = getEnvAsInt("PORT", 8080)
	config.DbConnection = getEnv("DB_CONNECTION", "mongodb://localhost:27017")
	config.DbNameSeed = getEnv("DB_NAME_SEED", "")
	config.GoogleClientId = getEnv("GOOGLE_CLIENT_ID", "")
	config.MicrosoftClientId = getEnv("MICROSOFT_CLIENT_ID", "")
	config.MicrosoftTenantId = getEnv("MICROSOFT_TENANT_ID", "")
	config.GithubAppName = getEnv("GITHUB_APP_NAME", "")
	config.Domain = getEnv("DOMAIN", "")
	config.DomainVerify = getEnv("DOMAIN_VERIFY", "false") == "true"
	if config.Domain == "" {
		return nil, errors.New("DOMAIN environment variable is required")
	}
	config.WebsiteName = getEnv("WEBSITE_NAME", "")
	if config.WebsiteName == "" {
		return nil, errors.New("WEBSITE_NAME environment variable is required")
	}
	config.EmailFromAddress = getEnv("EMAIL_FROM_ADDRESS", "")
	if config.EmailFromAddress == "" {
		return nil, errors.New("EMAIL_FROM_ADDRESS environment variable is required")
	}
	config.EmailFromAddress = strings.TrimSpace(config.EmailFromAddress)
	config.EnableSmtp = getEnv("ENABLE_SMTP", "false") == "true"
	config.TestMode = getEnv("TEST_MODE", "false") == "true"

	if config.EnableSmtp {
		config.EmailSmtpHost = getEnv("EMAIL_SMTP_HOST", "")
		if config.EmailSmtpHost == "" {
			return nil, errors.New("EMAIL_SMTP_HOST environment variable is required when SMTP is enabled")
		}
		config.EmailSmtpPort = getEnv("EMAIL_SMTP_PORT", "25")
		config.EmailSmtpUser = getEnv("EMAIL_SMTP_USER", "")
		config.EmailSmtpPass = getEnv("EMAIL_SMTP_PASS", "")
		config.EmailSmtpSecure = getEnv("EMAIL_SMTP_SECURE", "false") == "true"
		config.EmailAuth = getEnv("EMAIL_SMTP_AUTH", "false") == "true"
	}
	config.EmailTemplatesDir = getEnv("EMAIL_TEMPLATES_DIR", "")
	config.VerificationUrl = getEnv("VERIFICATION_URL", "")
	if config.VerificationUrl == "" {
		return nil, errors.New("VERIFICATION_URL environment variable is required")
	}
	config.ForgotPasswordUrl = getEnv("FORGOT_PASSWORD_URL", "")
	if config.ForgotPasswordUrl == "" {
		return nil, errors.New("FORGOT_PASSWORD_URL environment variable is required")
	}
	config.MagicUrl = getEnv("MAGIC_URL", "")
	if config.MagicUrl == "" {
		return nil, errors.New("MAGIC_URL environment variable is required")
	}
	config.MagicCodeExpireInMinutes = getEnvAsInt("MAGIC_CODE_EXPIRE_IN_MINUTES", 10)
	config.AccessTokenExpireInSeconds = getEnvAsInt("ACCESS_TOKEN_EXPIRE_IN_SECONDS", 3600)
	config.RefreshTokenExpireInSeconds = getEnvAsInt("REFRESH_TOKEN_EXPIRE_IN_SECONDS", 86400)
	config.AllowedOrigins = getEnvAsStringSlice("ALLOWED_WEB_ORIGINS", []string{})
	config.CompanyID = getEnv("COMPANY_ID", "")
	config.ApiKeyEnabled = getEnv("API_KEY_ENABLED", "false") == "true"
	config.CORSEnabled = getEnv("CORS_ENABLED", "false") == "true"

	return config, nil
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
