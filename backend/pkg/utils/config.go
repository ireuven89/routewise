package utils

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server                   Server   `mapstructure:"server"`
	Database                 Database `mapstructure:"database"`
	GoogleMapsAPIKey         string   `mapstructure:"google_maps_api_key"`
	GoogleMapsFrontendAPIKey string   `mapstructure:"google_maps_frontend_api_key"`
}

type Server struct {
	Port string `mapstructure:"port"`
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"user_name"`
	Password string `mapstructure:"password"`
}

func LoadConfig() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	v := viper.New()

	v.SetConfigName(env)
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("../config")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := v.BindEnv("database.password", "DB_PASSWORD"); err != nil {
		return nil, fmt.Errorf("failed to bind database config: %w", err)
	}
	if err := v.BindEnv("jwt.secret", "JWT_SECRET"); err != nil {
		return nil, fmt.Errorf("failed to bind jwt config: %w", err)
	}
	if err := v.BindEnv("google_maps_api_key", "GOOGLE_MAPS_API_KEY"); err != nil {
		return nil, fmt.Errorf("failed to bind google maps api key: %w", err)
	}
	if err := v.BindEnv("google_maps_frontend_api_key", "GOOGLE_MAPS_FRONTEND_API_KEY"); err != nil {
		return nil, fmt.Errorf("failed to bind google maps frontend api key: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
