package utils

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx mongo.SessionContext) error) error
}

type MongoTxManager struct {
	client *mongo.Client
}

func NewMongoTxManager(client *mongo.Client) *MongoTxManager {
	return &MongoTxManager{client: client}
}

func (m *MongoTxManager) WithTransaction(ctx context.Context, fn func(ctx mongo.SessionContext) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sessionCtx mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}

		if err := fn(sessionCtx); err != nil {
			_ = session.AbortTransaction(sessionCtx)
			return err
		}

		return session.CommitTransaction(sessionCtx)
	})
}
