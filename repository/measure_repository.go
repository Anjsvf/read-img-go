package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Anjsvf/read-img-go/config"
	"github.com/Anjsvf/read-img-go/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MeasureRepository interface {
	ExistsByTypeAndMonth(ctx context.Context, customerCode string, mt domain.MeasureType, ref time.Time) (bool, error)
	Create(ctx context.Context, m *domain.Measure) error
	FindByUUID(ctx context.Context, uuid string) (*domain.Measure, error)
	Confirm(ctx context.Context, uuid string, confirmedValue int) error
	ListByCustomer(ctx context.Context, customerCode string, mt *domain.MeasureType) ([]domain.Measure, error)
}

type mongoRepo struct {
	col *mongo.Collection
}

func NewMeasureRepository(cfg *config.Config, client *mongo.Client) (MeasureRepository, error) {
	col := client.Database(cfg.MongoDB).Collection(cfg.MongoCollection)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "customer_code", Value: 1},
			{Key: "measure_type", Value: 1},
			{Key: "measure_datetime", Value: 1},
		},
		Options: options.Index().SetSparse(true),
	}
	if _, err := col.Indexes().CreateOne(ctx, indexModel); err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	return &mongoRepo{col: col}, nil
}

func (r *mongoRepo) ExistsByTypeAndMonth(ctx context.Context, customerCode string, mt domain.MeasureType, ref time.Time) (bool, error) {
	start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	filter := bson.M{
		"customer_code": customerCode,
		"measure_type":  string(mt),
		"measure_datetime": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	count, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *mongoRepo) Create(ctx context.Context, m *domain.Measure) error {
	m.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, m)
	return err
}

func (r *mongoRepo) FindByUUID(ctx context.Context, uuid string) (*domain.Measure, error) {
	var m domain.Measure
	err := r.col.FindOne(ctx, bson.M{"measure_uuid": uuid}).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &m, err
}

func (r *mongoRepo) Confirm(ctx context.Context, uuid string, confirmedValue int) error {
	update := bson.M{
		"$set": bson.M{
			"confirmed_value": confirmedValue,
			"has_confirmed":   true,
		},
	}
	_, err := r.col.UpdateOne(ctx, bson.M{"measure_uuid": uuid}, update)
	return err
}

func (r *mongoRepo) ListByCustomer(ctx context.Context, customerCode string, mt *domain.MeasureType) ([]domain.Measure, error) {
	filter := bson.M{"customer_code": customerCode}
	if mt != nil {
		filter["measure_type"] = strings.ToUpper(string(*mt))
	}

	cursor, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var measures []domain.Measure
	if err := cursor.All(ctx, &measures); err != nil {
		return nil, err
	}
	return measures, nil
}
