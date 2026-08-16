// 配置加载（docs/02 §5：环境变量 > 默认值；缺失即报错）
package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	ServerAddr         string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	JWTSecret          string
	CORSAllowedOrigins []string
	// 登录会话有效期（JWT 与 Redis 会话键共用；env SESSION_TTL，如 2h/8h/24h，默认 2h）
	SessionTTL time.Duration
}

func Load() *Config {
	// 本地开发从当前工作目录读取 .env；已存在的环境变量优先，生产环境不隐式加载文件。
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}
	return &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		ServerAddr:    getEnv("SERVER_ADDR", ":8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		SessionTTL:    getEnvDuration("SESSION_TTL", 2*time.Hour),
		CORSAllowedOrigins: splitCSV(getEnv(
			"CORS_ALLOWED_ORIGINS",
			"http://localhost:5173,http://127.0.0.1:5173",
		)),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		if v := strings.TrimSpace(item); v != "" {
			out = append(out, v)
		}
	}
	return out
}
