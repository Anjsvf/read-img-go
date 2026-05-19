package handler

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Anjsvf/read-img-go/domain"
	"github.com/Anjsvf/read-img-go/service"
	"github.com/gin-gonic/gin"
)

type MeasureHandler struct {
	svc service.MeasureService
}

func NewMeasureHandler(svc service.MeasureService) *MeasureHandler {
	return &MeasureHandler{svc: svc}
}

func apiError(c *gin.Context, status int, code, description string) {
	c.JSON(status, domain.APIError{
		ErrorCode:        code,
		ErrorDescription: description,
	})
}

// POST /upload
func (h *MeasureHandler) Upload(c *gin.Context) {
	// Pega o customer_code do token JWT automaticamente
	customerCodeRaw, exists := c.Get("customer_code")
	if !exists {
		apiError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token inválido")
		return
	}
	customerCode := customerCodeRaw.(string)

	var base64Image string
	var measureDatetime, measureTypeStr string

	if strings.Contains(c.ContentType(), "multipart/form-data") {
		measureDatetime = c.PostForm("measure_datetime")
		measureTypeStr = c.PostForm("measure_type")

		file, err := c.FormFile("image")
		if err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_DATA", "image é obrigatória")
			return
		}

		src, err := file.Open()
		if err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_DATA", "erro ao ler imagem")
			return
		}
		defer src.Close()

		imgBytes, err := io.ReadAll(src)
		if err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_DATA", "erro ao processar imagem")
			return
		}

		base64Image = base64.StdEncoding.EncodeToString(imgBytes)

	} else {
		var raw struct {
			Image           string `json:"image"`
			MeasureDatetime string `json:"measure_datetime"`
			MeasureType     string `json:"measure_type"`
		}
		if err := c.ShouldBindJSON(&raw); err != nil {
			apiError(c, http.StatusBadRequest, "INVALID_DATA", err.Error())
			return
		}
		base64Image = raw.Image
		measureDatetime = raw.MeasureDatetime
		measureTypeStr = raw.MeasureType
	}

	if strings.TrimSpace(base64Image) == "" {
		apiError(c, http.StatusBadRequest, "INVALID_DATA", "image é obrigatória")
		return
	}
	if strings.TrimSpace(measureDatetime) == "" {
		apiError(c, http.StatusBadRequest, "INVALID_DATA", "measure_datetime é obrigatório")
		return
	}

	mt := domain.MeasureType(strings.ToUpper(measureTypeStr))
	if !mt.IsValid() {
		apiError(c, http.StatusBadRequest, "INVALID_DATA", "measure_type must be WATER or GAS")
		return
	}

	parsedTime, err := time.Parse(time.RFC3339, measureDatetime)
	if err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_DATA", "measure_datetime must be RFC3339 (ex: 2024-08-01T10:00:00Z)")
		return
	}

	req := domain.UploadRequest{
		Image:           base64Image,
		CustomerCode:    customerCode, // vem do JWT, não do body
		MeasureDatetime: parsedTime,
		MeasureType:     mt,
	}

	resp, err := h.svc.Upload(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDoubleReport):
			apiError(c, http.StatusConflict, "DOUBLE_REPORT", "Leitura do mês já realizada")
		case errors.Is(err, service.ErrInvalidBase64):
			apiError(c, http.StatusBadRequest, "INVALID_DATA", "image must be a valid base64 encoded string")
		default:
			apiError(c, http.StatusBadRequest, "INVALID_DATA", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PATCH /confirm
func (h *MeasureHandler) Confirm(c *gin.Context) {
	var req domain.ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiError(c, http.StatusBadRequest, "INVALID_DATA", err.Error())
		return
	}

	if strings.TrimSpace(req.MeasureUUID) == "" {
		apiError(c, http.StatusBadRequest, "INVALID_DATA", "measure_uuid is required")
		return
	}

	if err := h.svc.Confirm(c.Request.Context(), &req); err != nil {
		switch {
		case errors.Is(err, service.ErrMeasureNotFound):
			apiError(c, http.StatusNotFound, "MEASURE_NOT_FOUND", "Leitura não encontrada")
		case errors.Is(err, service.ErrConfirmationDuplicate):
			apiError(c, http.StatusConflict, "CONFIRMATION_DUPLICATE", "Leitura do mês já realizada")
		default:
			apiError(c, http.StatusBadRequest, "INVALID_DATA", err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, domain.ConfirmResponse{Success: true})
}

// GET /measures/list
func (h *MeasureHandler) List(c *gin.Context) {
	// customer_code vem do token JWT — usuário só vê as próprias leituras
	customerCodeRaw, exists := c.Get("customer_code")
	if !exists {
		apiError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token inválido")
		return
	}
	customerCode := customerCodeRaw.(string)

	var measureType *domain.MeasureType
	if rawType := c.Query("measure_type"); rawType != "" {
		mt := domain.MeasureType(strings.ToUpper(rawType))
		if !mt.IsValid() {
			apiError(c, http.StatusBadRequest, "INVALID_TYPE", "Tipo de medição não permitida")
			return
		}
		measureType = &mt
	}

	resp, err := h.svc.List(c.Request.Context(), customerCode, measureType)
	if err != nil {
		if errors.Is(err, service.ErrMeasuresNotFound) {
			apiError(c, http.StatusNotFound, "MEASURES_NOT_FOUND", "Nenhuma leitura encontrada")
			return
		}
		apiError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, resp)
}
