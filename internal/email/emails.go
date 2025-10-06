package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
)

const (
	verificationTemplate = "verification.html"
	forgotTemplate       = "forgot.html"
	magicTemplate        = "magic.html"
)

// Verification data for verification emails
type Verification struct {
	Email  string
	Token  string
	URL    string
	Domain string
}

type Forgot struct {
	Email  string
	Token  string
	URL    string
	Domain string
}

type Magic struct {
	Email  string
	Code   string
	URL    string
	Domain string
}

// EmailOptions for email server connections
type EmailOptions struct {
	VerificationUrl string
	ForgotUrl       string
	MagicUrl        string
	Auth            bool
	Tls             bool
	TestMode        bool
}

// EmailService email service contract
type EmailService interface {
	SendVerificationEmail(ctx context.Context, email, verificationToken string) error
	SendForgotPasswordEmail(ctx context.Context, email, forgotToken string) error
	SendMagicLinkEmail(ctx context.Context, email, code string) error
}

type EmailTemplateProvider interface {
	Initialize(ctx context.Context) error
}

// DefaultEmailService default email service
type DefaultEmailService struct {
	auth            smtp.Auth
	serverAddress   string
	port            string
	fromAddress     string
	baseUrl         string
	templatesDir    string
	domain          string
	emailRepository EmailRepository
	options         EmailOptions
}

// NewDefaultEmailService creates a an email service
func NewDefaultEmailService(user, secret, server, port, baseUrl, templatesDir, domain string, repository EmailRepository, options EmailOptions) *DefaultEmailService {
	auth := smtp.PlainAuth("", user, secret, server)
	return &DefaultEmailService{
		auth:            auth,
		serverAddress:   server,
		port:            port,
		emailRepository: repository,
		fromAddress:     "no-reply@" + domain,
		baseUrl:         baseUrl,
		templatesDir:    templatesDir,
		options:         options,
	}
}

func (s *DefaultEmailService) Initialize(ctx context.Context) error {
	err := s.verification(ctx)
	if err != nil {
		return err
	}
	err = s.forgot(ctx)
	if err != nil {
		return err
	}
	err = s.magic(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *DefaultEmailService) verification(ctx context.Context) error {
	templateFile, err := os.ReadFile(fmt.Sprintf("%s%s", s.templatesDir, verificationTemplate))
	if err != nil {
		return err
	}

	t, err := s.emailRepository.Read(ctx, "verification")
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}
	}

	if t == "" {
		err := s.emailRepository.Create(ctx, "verification", string(templateFile))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DefaultEmailService) forgot(ctx context.Context) error {
	templateFile, err := os.ReadFile(fmt.Sprintf("%s%s", s.templatesDir, forgotTemplate))
	if err != nil {
		return err
	}

	t, err := s.emailRepository.Read(ctx, "forgot")
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}
	}

	if t == "" {
		err := s.emailRepository.Create(ctx, "forgot", string(templateFile))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DefaultEmailService) magic(ctx context.Context) error {
	templateFile, err := os.ReadFile(fmt.Sprintf("%s%s", s.templatesDir, magicTemplate))
	if err != nil {
		return err
	}

	t, err := s.emailRepository.Read(ctx, "magic")
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}
	}

	if t == "" {
		err := s.emailRepository.Create(ctx, "magic", string(templateFile))
		if err != nil {
			return err
		}
	}

	return nil
}

// SendVerificationEmail will send out the verification email when a user signs up to activate their account
func (s *DefaultEmailService) SendVerificationEmail(ctx context.Context, email, verificationToken string) error {
	subject := "Please verify account"

	t, err := s.emailRepository.Read(ctx, "verification")

	if err != nil {
		return err
	}

	tmpl := template.Must(template.New("verification").Parse(t))

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, Verification{Email: email, Token: verificationToken, Domain: s.baseUrl, URL: s.options.VerificationUrl}); err != nil {
		return err
	}
	if s.options.TestMode {
		subject = verificationToken
	}
	msg := []byte("From: " + s.fromAddress + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
		buf.String())

	if err = smtp.SendMail(s.serverAddress+":"+s.port, s.auth, s.fromAddress, []string{email}, msg); err != nil {
		return err
	}

	return nil
}

func (s *DefaultEmailService) SendForgotPasswordEmail(ctx context.Context, email, forgotToken string) error {
	subject := "Password reset requested"
	t, err := s.emailRepository.Read(ctx, "forgot")

	if err != nil {
		return err
	}

	tmpl := template.Must(template.New("forgot").Parse(t))

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, Forgot{Email: email, Token: forgotToken, Domain: s.baseUrl, URL: s.options.ForgotUrl}); err != nil {
		return err
	}
	if s.options.TestMode {
		subject = forgotToken
	}

	msg := []byte("From: " + s.fromAddress + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
		buf.String())

	if err = smtp.SendMail(s.serverAddress+":"+s.port, s.auth, s.fromAddress, []string{email}, msg); err != nil {
		return err
	}

	return nil
}

func (s *DefaultEmailService) SendMagicLinkEmail(ctx context.Context, email, code string) error {
	subject := "Login link requested"
	t, err := s.emailRepository.Read(ctx, "magic")

	if err != nil {
		return err
	}

	tmpl := template.Must(template.New("magic").Parse(t))

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, Magic{Email: email, Code: code, Domain: s.baseUrl, URL: s.options.MagicUrl}); err != nil {
		return err
	}
	if s.options.TestMode {
		subject = code
	}

	msg := []byte("From: " + s.fromAddress + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" +
		buf.String())

	if err = smtp.SendMail(s.serverAddress+":"+s.port, s.auth, s.fromAddress, []string{email}, msg); err != nil {
		return err
	}

	return nil
}
