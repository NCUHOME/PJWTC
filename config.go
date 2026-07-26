package pjwt

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const defaultRequestTimeout = 2 * time.Second

var Addr = envOrDefault("PJWT_ADDR", "jwt-grpc.passport:80")

type Config struct {
	Addr           string
	TLS            bool
	CAFile         string
	ServerName     string
	RequestTimeout time.Duration
}

func configFromEnv() (Config, error) {
	config := Config{
		Addr:           Addr,
		CAFile:         os.Getenv("PJWT_CA_FILE"),
		ServerName:     os.Getenv("PJWT_SERVER_NAME"),
		RequestTimeout: defaultRequestTimeout,
	}

	if raw := os.Getenv("PJWT_TLS"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse PJWT_TLS: %w", err)
		}
		config.TLS = enabled
	}
	if raw := os.Getenv("PJWT_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("parse PJWT_TIMEOUT: invalid duration %q", raw)
		}
		config.RequestTimeout = timeout
	}
	return config, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
