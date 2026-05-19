package handler

import (
	"errors"
	"net/http"

	"github.com/Anjsvf/read-img-go/domain"
	"github.com/Anjsvf/read-img-go/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error_code":        "INVALID_DATA",
			"error_description": err.Error(),
		})
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error_code":        "EMAIL_ALREADY_EXISTS",
				"error_description": "Este email já está cadastrado",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error_code":        "INTERNAL_ERROR",
			"error_description": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error_code":        "INVALID_DATA",
			"error_description": err.Error(),
		})
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error_code":        "UNAUTHORIZED",
				"error_description": "Email ou senha incorretos",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error_code":        "INTERNAL_ERROR",
			"error_description": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
