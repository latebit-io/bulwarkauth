package tokens

import (
	"context"
)

type TokenRepository interface {
	DeleteByDeviceId(ctx context.Context, email string, deviceId string) error
	Delete(ctx context.Context, email string) error
	Acknowledge(ctx context.Context, email string, deviceId string, accessToken string, refreshToken string) error
	Get(ctx context.Context, email string, deviceId string) (AuthToken, error)
}
