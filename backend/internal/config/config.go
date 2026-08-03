package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	ServerPort     string
	DatabaseURL    string
	RedisURL       string
	AllowedOrigins []string
	R2Endpoint  string
	R2AccessKey string
	R2SecretKey string
	R2Bucket    string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		DatabaseURL:    databaseURL(),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AllowedOrigins: allowedOrigins(),
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

// allowedOrigins lists the browser origins the API answers. The default covers
// local development only — the Vite dev server and the Compose stack — so any
// other deployment must name its own origin, scheme and port included.
func allowedOrigins() []string {
	raw := getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:6900")

	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		log.Fatal("ALLOWED_ORIGINS was set but contained no origins")
	}
	return out
}

// databaseURL reads the Postgres connection string. Supabase's dashboard offers
// several connection variants across separate tabs, and pasting the wrong one
// surfaces as an opaque driver error — so check the scheme here, where we can
// name what is expected.
func databaseURL() string {
	v := mustEnv("DATABASE_URL")
	if !strings.HasPrefix(v, "postgres://") && !strings.HasPrefix(v, "postgresql://") {
		log.Fatal("DATABASE_URL must be a postgres:// or postgresql:// connection string, " +
			"e.g. the Session pooler URI from the Supabase dashboard")
	}
	return v
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
