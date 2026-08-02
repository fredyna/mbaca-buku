package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	ServerPort  string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	RedisURL    string
	R2Endpoint  string
	R2AccessKey string
	R2SecretKey string
	R2Bucket    string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "mbaca"),
		DBPassword:  getEnv("DB_PASSWORD", "mbaca_secret"),
		DBName:      getEnv("DB_NAME", "mbaca_buku"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		R2Endpoint:  r2Endpoint(),
		R2AccessKey: mustEnv("R2_ACCESS_KEY_ID"),
		R2SecretKey: mustEnv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:    mustEnv("R2_BUCKET"),
		JWTSecret:   mustEnv("JWT_SECRET"),
	}
}

// r2Endpoint reads the R2 host. Cloudflare's dashboard presents the S3 API as
// a full URL with the bucket appended, but both the S3 client and the nginx
// proxy template need the bare host — so reject the pasted form here, where we
// can say exactly what to strip, rather than let it surface as an opaque SDK
// error at startup or a broken proxy_pass in nginx.
func r2Endpoint() string {
	v := mustEnv("R2_ENDPOINT")
	if strings.Contains(v, "://") || strings.Contains(v, "/") {
		log.Fatalf("R2_ENDPOINT must be the host only, e.g. <account-id>.r2.cloudflarestorage.com "+
			"— drop the https:// prefix and any /<bucket> suffix (the bucket belongs in R2_BUCKET); got %q", v)
	}
	return v
}

func (c *Config) DBDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

// mustEnv reads a setting that has no sensible default. R2 credentials and the
// JWT secret are only ever correct when supplied by the environment, so a
// missing one is a misconfiguration worth failing on at startup rather than at
// the first upload.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
