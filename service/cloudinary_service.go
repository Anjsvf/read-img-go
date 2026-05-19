package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Anjsvf/read-img-go/config"
)

type CloudinaryService interface {
	Upload(ctx context.Context, base64Image string) (string, error)
}

type cloudinarySvc struct {
	cloudName    string
	uploadPreset string
}

func NewCloudinaryService(cfg *config.Config) CloudinaryService {
	return &cloudinarySvc{
		cloudName:    cfg.CloudinaryCloud,
		uploadPreset: cfg.CloudinaryFolder, // usamos o folder como nome do preset
	}
}

type cloudinaryUploadResponse struct {
	SecureURL string `json:"secure_url"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *cloudinarySvc) Upload(ctx context.Context, base64Image string) (string, error) {
	if idx := strings.Index(base64Image, ","); idx != -1 {
		base64Image = base64Image[idx+1:]
	}

	imgBytes, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", c.cloudName)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "meter.jpg")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imgBytes)); err != nil {
		return "", fmt.Errorf("write image bytes: %w", err)
	}

	_ = writer.WriteField("upload_preset", c.uploadPreset)

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloudinary request: %w", err)
	}
	defer resp.Body.Close()

	var result cloudinaryUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode cloudinary response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("cloudinary error: %s", result.Error.Message)
	}

	return result.SecureURL, nil
}
