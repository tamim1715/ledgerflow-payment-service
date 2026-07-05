package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	v := viper.New()

	// Config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath("../configs")
	v.AddConfigPath("../../configs")

	// Environment variables
	v.SetEnvPrefix("LEDGERFLOW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read config file: %w", err)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	return &cfg, nil
}
