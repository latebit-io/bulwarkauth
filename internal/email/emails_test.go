package email

import (
	"context"
	"os"
)

type MockEmailRepository struct {
}

func NewMockEmailRepository() *MockEmailRepository {
	return &MockEmailRepository{}
}

func (r *MockEmailRepository) Read(ctx context.Context, name string) (string, error) {
	content, err := os.ReadFile("../../verification.html")
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (r *MockEmailRepository) Create(ctx context.Context, name, template string) error {
	return nil
}

func (r *MockEmailRepository) Update(ctx context.Context, name, template string) error {
	return nil
}

//func TestDefaultEmailRepository_SendVerification(t *testing.T) {
//	repo := NewMockEmailRepository()
//	emailService := NewDefaultEmailService("test", "test", "localhost", "1025", "latebit.io", "/", "latebit.io", repo, EmailOptions{})
//	err := emailService.SendVerificationEmail(context.Background(), "test@latebit.io", "verification")
//	if err != nil {
//		t.Fatal(err)
//	}
//}
