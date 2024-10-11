package configs

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type DomainsConfig struct {
	// Server configs
	ServeAddress string `mapstructure:"SERVE_ADDRESS"`

	// Aws S3
	AwsAccessKeyId        string `mapstructure:"AWS_ACCESS_KEY_ID"`
	AwsSecretAccessKey    string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	ScanResultsBucketName string `mapstructure:"SCAN_RESULTS_BUCKET_NAME"`
	TemplateBucketName    string `mapstructure:"TEMPLATE_BUCKET_NAME"`

	// MongoDB related configs
	MongoDbUri  string `mapstructure:"MONGO_DB_URI"`
	MongoDbName string `mapstructure:"MONGO_DB_NAME"`

	// RabbitMQ related configs
	RabbitMqUri string `mapstructure:"RABBIT_MQ_URI"`
}

func LoadDomainsConfig(path string) (DomainsConfig, error) {
	if path == "" {
		return DomainsConfig{}, fmt.Errorf("config path is empty")
	}

	viper.AutomaticEnv()

	// This is for local development
	viper.AddConfigPath(path)
	viper.SetConfigName(".env.domains")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Info().Msg("Loading env from os environment variables")
		} else {
			// Config file was found but another error was produced
			return DomainsConfig{}, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config DomainsConfig
	if err := viper.Unmarshal(&config); err != nil {
		return DomainsConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
