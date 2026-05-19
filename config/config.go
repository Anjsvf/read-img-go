package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	MongoURI         string
	MongoDB          string
	MongoCollection  string
	UsersCollection  string
	GeminiAPIKey     string
	CloudinaryURL    string
	CloudinaryCloud  string
	CloudinaryAPIKey string
	CloudinarySecret string
	CloudinaryFolder string
	JWTSecret        string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		Port:             getEnv("PORT", "8080"),
		MongoURI:         requireEnv("MONGO_URI"),
		MongoDB:          getEnv("MONGO_DB", "shopper"),
		MongoCollection:  getEnv("MONGO_COLLECTION", "measures"),
		UsersCollection:  getEnv("USERS_COLLECTION", "users"),
		GeminiAPIKey:     requireEnv("GEMINI_API_KEY"),
		CloudinaryURL:    requireEnv("CLOUDINARY_URL"),
		CloudinaryCloud:  requireEnv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey: requireEnv("CLOUDINARY_API_KEY"),
		CloudinarySecret: requireEnv("CLOUDINARY_API_SECRET"),
		CloudinaryFolder: getEnv("CLOUDINARY_FOLDER", "shopper-meters"),
		JWTSecret:        requireEnv("JWT_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Required environment variable %q is not set", key)
	}
	return v
}
