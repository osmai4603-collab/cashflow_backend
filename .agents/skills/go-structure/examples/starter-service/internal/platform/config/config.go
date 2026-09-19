package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config إعدادات التطبيق المجمعة
type Config struct {
	Port            string
	Environment     string
	ShutdownTimeout time.Duration
}

// Load تحميل الإعدادات من متغيرات البيئة مع قيم افتراضية آمنة وفحص صارم
func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid PORT value '%s': must be numeric", port)
	}

	env := getEnv("APP_ENV", "development")
	timeoutSec, _ := strconv.Atoi(getEnv("SHUTDOWN_TIMEOUT_SEC", "10"))

	return &Config{
		Port:            port,
		Environment:     env,
		ShutdownTimeout: time.Duration(timeoutSec) * time.Second,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}
