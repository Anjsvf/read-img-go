package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Anjsvf/read-img-go/config"
)

type GeminiService interface {
	ExtractMeterValue(ctx context.Context, base64Image string) (int, error)
}

type geminiSvc struct {
	apiKey string
}

func NewGeminiService(cfg *config.Config) GeminiService {
	return &geminiSvc{apiKey: cfg.GeminiAPIKey}
}

// Gemini API request/response structures

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string        `json:"text,omitempty"`
	InlineData *geminiInline `json:"inline_data,omitempty"`
}

type geminiInline struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ExtractMeterValue sends the base64 image to Gemini Vision and parses the numeric reading.
// The base64 string is transmitted directly to the Gemini API and is not stored anywhere.
func (g *geminiSvc) ExtractMeterValue(ctx context.Context, base64Image string) (int, error) {
	// Strip data URI prefix if present
	mimeType := "image/jpeg"
	if idx := strings.Index(base64Image, ";base64,"); idx != -1 {
		mimeStart := strings.Index(base64Image, "data:")
		if mimeStart != -1 {
			mimeType = base64Image[mimeStart+5 : idx]
		}
		base64Image = base64Image[idx+8:]
	}

	payload := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{
						Text: "You are a meter reading assistant. Analyze this image of a water or gas meter. " +
							"Extract ONLY the numeric value shown on the meter display. " +
							"Respond with ONLY the integer number, no units, no text, no punctuation. " +
							"If you cannot determine the value, respond with 0.",
					},
					{
						InlineData: &geminiInline{
							MimeType: mimeType,
							Data:     base64Image,
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-latest:generateContent?key=%s",
		g.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode gemini response: %w", err)
	}

	if result.Error != nil {
		return 0, fmt.Errorf("gemini error: %s", result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return 0, fmt.Errorf("gemini returned no candidates")
	}

	rawText := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	rawText = strings.ReplaceAll(rawText, ",", "")
	rawText = strings.ReplaceAll(rawText, ".", "")

	var value int
	if _, err := fmt.Sscanf(rawText, "%d", &value); err != nil {
		return 0, fmt.Errorf("could not parse meter value from gemini response %q: %w", rawText, err)
	}

	return value, nil
}
