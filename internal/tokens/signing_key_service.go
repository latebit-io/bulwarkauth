package tokens

import "context"

type SigningKeyService interface {
	GenerateKey(ctx context.Context) error
	GetAllKeys(ctx context.Context) ([]SigningKey, error)
	LatestKey(ctx context.Context) (*SigningKey, error)
	Initialize(ctx context.Context) error
}

type DefaultSigningKeyService struct {
	repository SigningKeyRepository
}

func NewDefaultSigningKeyService(repository SigningKeyRepository) *DefaultSigningKeyService {
	return &DefaultSigningKeyService{
		repository: repository,
	}
}

func (s *DefaultSigningKeyService) GenerateKey(ctx context.Context) error {
	key, err := NewSigningKey(256)
	if err != nil {
		return err
	}

	err = s.repository.Add(ctx, key.PrivateKey, key.PublicKey, key.Algorithm)
	if err != nil {
		return err
	}

	return nil
}

func (s *DefaultSigningKeyService) GetAllKeys(ctx context.Context) ([]SigningKey, error) {
	return s.repository.GetAllKeys(ctx)
}

func (s *DefaultSigningKeyService) Initialize(ctx context.Context) error {
	latest, err := s.repository.GetLatestKey(ctx)
	if err != nil {
		return err
	}

	if latest.Algorithm != "" {
		return nil
	}

	key, err := NewSigningKey(256)
	if err != nil {
		return err
	}

	return s.repository.Add(ctx, key.PrivateKey, key.PublicKey, key.Algorithm)
}

func (s *DefaultSigningKeyService) LatestKey(ctx context.Context) (*SigningKey, error) {
	latest, err := s.repository.GetLatestKey(ctx)
	if err != nil {
		return nil, err
	}
	return &latest, nil
}
