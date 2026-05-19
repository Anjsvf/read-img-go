package router

import (
	"context"
	"log"
	"time"

	"github.com/Anjsvf/read-img-go/config"
	"github.com/Anjsvf/read-img-go/handler"
	"github.com/Anjsvf/read-img-go/middleware"
	"github.com/Anjsvf/read-img-go/repository"
	"github.com/Anjsvf/read-img-go/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Setup(cfg *config.Config) *gin.Engine {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	measureRepo, err := repository.NewMeasureRepository(cfg, client)
	if err != nil {
		log.Fatalf("Failed to setup measure repository: %v", err)
	}
	userRepo, err := repository.NewUserRepository(cfg, client)
	if err != nil {
		log.Fatalf("Failed to setup user repository: %v", err)
	}

	geminiSvc := service.NewGeminiService(cfg)
	cloudinarySvc := service.NewCloudinaryService(cfg)
	measureSvc := service.NewMeasureService(measureRepo, geminiSvc, cloudinarySvc)
	authSvc := service.NewAuthService(userRepo, cfg)

	measureHandler := handler.NewMeasureHandler(measureSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS — permite o frontend acessar a API
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://read-img-front.vercel.app/"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(middleware.Security())
	r.Use(middleware.MaxBodySize(20 * 1024 * 1024))
	r.Use(middleware.RequestTimeout(60 * time.Second))
	r.Use(middleware.RateLimit())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)

	protected := r.Group("/")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.POST("/upload", measureHandler.Upload)
		protected.PATCH("/confirm", measureHandler.Confirm)
		protected.GET("/measures/list", measureHandler.List)
	}

	return r
}