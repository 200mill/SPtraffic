package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Redis    RedisConfig
	DataGokr DataGokrConfig
	Admin    AdminConfig
}

type ServerConfig struct {
	Port          string
	CORSOrigin    string
	MigrationsDir string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

func (d DBConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + d.Port +
		" dbname=" + d.Name +
		" user=" + d.User +
		" password=" + d.Password +
		" sslmode=disable"
}

type RedisConfig struct {
	Addr string
}

type DataGokrConfig struct {
	APIKey string
}

// AdminConfig holds credentials for the /api/admin/* endpoints.
type AdminConfig struct {
	// Key is the secret value required in the X-Admin-Key header.
	// If empty, admin endpoints are disabled.
	Key string
}

func Load() *Config {
	// Optional .env file — silently ignored if absent (Docker injects env vars directly)
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("[config] no .env file loaded (%v) — using environment variables", err)
	}

	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("CORS_ORIGIN", "http://localhost:5173")
	viper.SetDefault("MIGRATIONS_DIR", "./migrations")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_NAME", "sptraffic")
	viper.SetDefault("DB_USER", "sptraffic")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("DATAGOKR_API_KEY", "")
	viper.SetDefault("ADMIN_KEY", "")

	return &Config{
		Server: ServerConfig{
			Port:          viper.GetString("SERVER_PORT"),
			CORSOrigin:    viper.GetString("CORS_ORIGIN"),
			MigrationsDir: viper.GetString("MIGRATIONS_DIR"),
		},
		DB: DBConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			Name:     viper.GetString("DB_NAME"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
		},
		Redis: RedisConfig{
			Addr: viper.GetString("REDIS_ADDR"),
		},
		DataGokr: DataGokrConfig{
			APIKey: viper.GetString("DATAGOKR_API_KEY"),
		},
		Admin: AdminConfig{
			Key: viper.GetString("ADMIN_KEY"),
		},
	}
}
