package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MeasureType string

const (
	MeasureTypeWater MeasureType = "WATER"

	MeasureTypeGas MeasureType = "GAS"
)

func (m MeasureType) IsValid() bool {
	return m == MeasureTypeWater || m == MeasureTypeGas
}

type Measure struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"  json:"-"`
	MeasureUUID     string             `bson:"measure_uuid"   json:"measure_uuid"`
	CustomerCode    string             `bson:"customer_code"  json:"customer_code"`
	MeasureType     MeasureType        `bson:"measure_type"   json:"measure_type"`
	MeasureDatetime time.Time          `bson:"measure_datetime" json:"measure_datetime"`
	ImageURL        string             `bson:"image_url"      json:"image_url"`
	MeasureValue    int                `bson:"measure_value"  json:"measure_value"`
	ConfirmedValue  *int               `bson:"confirmed_value,omitempty" json:"confirmed_value,omitempty"`
	HasConfirmed    bool               `bson:"has_confirmed"  json:"has_confirmed"`
	CreatedAt       time.Time          `bson:"created_at"     json:"-"`
}

type UploadRequest struct {
	Image           string      `json:"image"            binding:"required"`
	CustomerCode    string      `json:"customer_code"    binding:"required"`
	MeasureDatetime time.Time   `json:"measure_datetime" binding:"required"`
	MeasureType     MeasureType `json:"measure_type"     binding:"required"`
}

type UploadResponse struct {
	ImageURL     string `json:"image_url"`
	MeasureValue int    `json:"measure_value"`
	MeasureUUID  string `json:"measure_uuid"`
}

type ConfirmRequest struct {
	MeasureUUID    string `json:"measure_uuid"     binding:"required"`
	ConfirmedValue int    `json:"confirmed_value"  binding:"required"`
}

type ConfirmResponse struct {
	Success bool `json:"success"`
}

type MeasureListItem struct {
	MeasureUUID     string      `json:"measure_uuid"`
	MeasureDatetime time.Time   `json:"measure_datetime"`
	MeasureType     MeasureType `json:"measure_type"`
	HasConfirmed    bool        `json:"has_confirmed"`
	ImageURL        string      `json:"image_url"`
}

type ListResponse struct {
	CustomerCode string            `json:"customer_code"`
	Measures     []MeasureListItem `json:"measures"`
}

type APIError struct {
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
}
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"  json:"-"`
	CustomerCode string             `bson:"customer_code"  json:"customer_code"`
	Name         string             `bson:"name"           json:"name"`
	Email        string             `bson:"email"          json:"email"`
	Password     string             `bson:"password"       json:"-"`
	CreatedAt    time.Time          `bson:"created_at"     json:"-"`
}

type RegisterRequest struct {
	Name     string `json:"name"     binding:"required"`
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token        string `json:"token"`
	ExpiresIn    string `json:"expires_in"`
	CustomerCode string `json:"customer_code"`
	Name         string `json:"name"`
}
