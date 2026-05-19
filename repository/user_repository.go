package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Anjsvf/read-img-go/config"
	"github.com/Anjsvf/read-img-go/domain"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

type mongoUserRepo struct {
	col *mongo.Collection
}

func NewUserRepository(cfg *config.Config, client *mongo.Client) (UserRepository, error) {
	col := client.Database(cfg.MongoDB).Collection(cfg.UsersCollection)

	// índice único no email
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("create user index: %w", err)
	}

	return &mongoUserRepo{col: col}, nil
}

func (r *mongoUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &user, err
}

func (r *mongoUserRepo) Create(ctx context.Context, user *domain.User) error {
	user.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, user)
	return err
}
