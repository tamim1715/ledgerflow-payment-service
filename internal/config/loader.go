package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	// Load .env file into OS environment if present
	_ = godotenv.Load(".env")

	v := viper.New()

	// Set default configuration values
	v.SetDefault("server.name", "ledgerflow-payment-service")
	v.SetDefault("server.port", 8081)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.shutdown_timeout", 10*time.Second)

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.name", "ledgerflow")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.max_idle_conns", 25)
	v.SetDefault("database.conn_max_lifetime", 30*time.Minute)
	v.SetDefault("database.conn_max_idle_time", 10*time.Minute)

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.development", true)

	// Configure environment variable binding with LEDGERFLOW prefix
	v.SetEnvPrefix("LEDGERFLOW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	return &cfg, nil
}
