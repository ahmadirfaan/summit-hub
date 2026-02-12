package config

import "github.com/spf13/viper"

type Config struct {
	ServerPort   string `mapstructure:"SERVER_PORT"`
	PostgresURL  string `mapstructure:"POSTGRES_URL"`
	RedisAddr    string `mapstructure:"REDIS_ADDR"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	JWTSecret    string `mapstructure:"JWT_SECRET"`
}

func Load() Config {
	viper.AutomaticEnv()
	viper.SetDefault("SERVER_PORT", ":8080")
	viper.SetDefault("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/summithub?sslmode=disable")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("JWT_SECRET", "dev-secret-change-me")

	var cfg Config
	_ = viper.Unmarshal(&cfg)
	return cfg
}
