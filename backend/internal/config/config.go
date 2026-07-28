package config

import (
	"log"
	"os"
)

type Config struct {
	ServerPort    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	RedisURL      string
	MinIOEndpoint string
	MinIOUser     string
	MinIOPassword string
	MinIOBucket   string
	MinIOUseSSL   bool
	JWTSecret     string
}

func Load() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "mbaca"),
		DBPassword:    getEnv("DB_PASSWORD", "mbaca_secret"),
		DBName:        getEnv("DB_NAME", "mbaca_buku"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MinIOEndpoint: getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOUser:     getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinIOPassword: getEnv("MINIO_ROOT_PASSWORD", "minioadmin123"),
		MinIOBucket:   getEnv("MINIO_BUCKET", "mbaca-buku"),
		MinIOUseSSL:   getEnv("MINIO_USE_SSL", "false") == "true",
		JWTSecret:     jwtSecret,
	}
}

func (c *Config) DBDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
