package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	accountsapi "github.com/latebit-io/bulwarkauth/api/accounts"
	authenticationapi "github.com/latebit-io/bulwarkauth/api/authentication"
	domainapi "github.com/latebit-io/bulwarkauth/api/domain"
	"github.com/latebit-io/bulwarkauth/api/health"
	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/authentication"
	"github.com/latebit-io/bulwarkauth/internal/authentication/social"
	"github.com/latebit-io/bulwarkauth/internal/domain"
	"github.com/latebit-io/bulwarkauth/internal/email"
	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/latebit-io/bulwarkauth/internal/version"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	// If the version flag is passed, print the version and exit
	if *versionFlag {
		fmt.Println(version.GetVersionInfo())
		os.Exit(0)
	}

	logger := getLogger()
	fmt.Println(`
 ____  _  _  __    _  _   __   ____  __ _   __   _  _  ____  _  _ 
(  _ \/ )( \(  )  / )( \ / _\ (  _ \(  / ) / _\ / )( \(_  _)/ )( \
 ) _ () \/ (/ (_/\\ /\ //    \ )   / )  ( /    \) \/ (  )(  ) __ (
(____/\____/\____/(_/\_)\_/\_/(__\_)(__\_)\_/\_/\____/ (__) \_)(_/ v1.0.0`)
	err := godotenv.Load()
	if err != nil {
		logger.Warn("no .env file loading from system")
	}

	config, err := NewAppConfig()
	if err != nil {
		panic(err)
	}

	service := echo.New()
	service.HideBanner = true
	logger.Info("connecting to mongodb: ", "uri", config.DbConnection, "db", config.DbNameSeed)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(config.DbConnection))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	mongodb := client.Database("bulwarkauth" + config.DbNameSeed)
	mongodbTxManager := utils.NewMongoTxManager(client)
	encrypt := encryption.NewDefaultEncryption()
	accountsRepo := accounts.NewMongodbAccountRepository(mongodb, encrypt)
	forgotRepo := accounts.NewMongoDbForgotRepository(mongodb)
	signingRepo := tokens.NewDefaultSigningKeyRepository(mongodb)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	err = signingService.Initialize(context.Background())
	if err != nil {
		panic(err)
	}
	tokenizer := tokens.NewDefaultTokenizer("bulwark-auth", "bulwark-auth", config.Domain,
		config.RefreshTokenExpireInSeconds, config.AccessTokenExpireInSeconds, signingService)
	emailRepo := email.NewMongoDbEmailRepository(mongodb)
	emailService := email.NewDefaultEmailService(config.EmailSmtpUser, config.EmailSmtpPass,
		config.EmailSmtpHost, config.EmailSmtpPort, config.Domain, config.EmailTemplatesDir, config.Domain, emailRepo, email.EmailOptions{
			VerificationUrl: config.VerificationUrl,
			ForgotUrl:       config.ForgotPasswordUrl,
			MagicUrl:        config.MagicUrl,
			TestMode:        config.TestMode,
		})
	wd, _ := os.Getwd()
	logger.Info("working directory: ", "dir", wd)
	err = emailService.Initialize(context.Background())
	if err != nil {
		panic(err)
	}
	accountsService := accounts.NewDefaultAccountService(accountsRepo, forgotRepo, tokenizer, emailService, mongodbTxManager)
	accountHandlers := accountsapi.NewAccountHandler(accountsService)
	accountsapi.AccountRoutes(service, accountHandlers)
	tokenRepo := authentication.NewDefaultTokenRepository(mongodb)
	authenticationService := authentication.NewDefaultAuthenticationService(accountsRepo, tokenRepo, tokenizer)
	authenticationHandler := authenticationapi.NewAuthenticationHandler(authenticationService)
	authenticationapi.AuthenticationRoutes(service, authenticationHandler)
	logonRepo := authentication.NewDefaultLogonCodeRepository(mongodb)
	logonService := authentication.NewDefaultLogonService(logonRepo, accountsRepo, emailService, tokenizer, encrypt)
	logonCodeHandlers := authenticationapi.NewLogonCodeHandlers(logonService)
	authenticationapi.LogonRoutes(service, logonCodeHandlers)
	google, err := social.NewGoogleValidator(config.GoogleClientId)
	if err != nil {
		panic(err)
	}
	socialService := social.NewDefaultSocialService(accountsRepo, accountsService, encrypt, tokenizer)
	socialService.AddValidator(google)
	socialHandlers := authenticationapi.NewSocialHandlers(socialService)
	authenticationapi.SocialRoutes(service, socialHandlers)

	if config.DomainVerify {
		domainRepo := domain.NewDefaultDomainRepository(mongodb)
		domainService := domain.NewDefaultDomainService(domainRepo, config.CompanyID)
		domains, err := domainService.GetAll(context.Background())
		if err != nil {
			panic(err)
		}

		for _, d := range domains {
			err = domainService.Verify(context.Background(), d.Domain)
			if err != nil {
				panic(err)
			}
		}
		domainHandlers := domainapi.NewDomainHandler(domainService)

		domainapi.DomainRoutes(service, domainHandlers)
	}
	corsSetting(service, config, logger)
	apiKeySetting(service, config, logger)

	healthHandler := health.NewHealthHandler()
	health.HealthRoutes(service, healthHandler)

	if err := service.Start(fmt.Sprintf(":%d", config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(err.Error())
	}
}

func getLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stderr, nil)
	logger := slog.New(jsonHandler)
	return logger
}

func corsSetting(service *echo.Echo, config *AppConfig, logger *slog.Logger) {
	if !config.CORSEnabled {
		return
	}
	config.AllowedOrigins = append(config.AllowedOrigins, fmt.Sprintf("https://%s", config.Domain))

	service.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: config.AllowedOrigins,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	logger.Info("cors enabled")
}

func apiKeySetting(service *echo.Echo, config *AppConfig, logger *slog.Logger) {
	if !config.ApiKeyEnabled {
		return
	}
	service.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:X-BULWARK-API-KEY",
		Validator: func(key string, c echo.Context) (bool, error) {
			return key == os.Getenv("API_KEY"), nil
		},
	}))
	logger.Info("api key enabled")
}
