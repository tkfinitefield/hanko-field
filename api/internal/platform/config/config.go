package config

import "os"

// AppConfig stores runtime configuration values pulled from environment variables.
type AppConfig struct {
	Port string
}

// Load reads application configuration from environment variables, providing sensible defaults
// so the service can run locally without manual setup.
func Load() (AppConfig, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return AppConfig{
		Port: port,
	}, nil
}
