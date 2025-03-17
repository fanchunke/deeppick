package config

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP   HTTP   `mapstructure:"http" structs:"http"`
	OpenAI OpenAI `mapstructure:"openai" structs:"openai"`
	Otel   Otel   `mapstructure:"otel" structs:"otel"`
	Cos    Cos    `mapstructure:"cos" structs:"cos"`
}

type HTTP struct {
	Port int `mapstructure:"port" structs:"port" env:"HTTP_PORT"`
}

type OpenAI struct {
	BaseUrl string `mapstructure:"base_url" structs:"base_url" env:"OPENAI_BASE_URL"`
	ApiKey  string `mapstructure:"api_key" structs:"api_key" env:"OPENAI_API_KEY"`
	Model   string `mapstructure:"model" structs:"model" env:"OPENAI_MODEL"`
}

type Otel struct {
	ServiceName       string `mapstructure:"service_name" structs:"service_name" env:"OTEL_SERVICE_NAME"`
	ServiceVersion    string `mapstructure:"service_version" structs:"service_version" env:"OTEL_SERVICE_VERSION"`
	DeployEnvironment string `mapstructure:"deploy_environment" structs:"deploy_environment" env:"OTEL_DEPLOY_ENVIRONMENT"`
	HTTPEndpoint      string `mapstructure:"http_endpoint" structs:"http_endpoint" env:"OTEL_HTTP_ENDPOINT"`
	HTTPUrlPath       string `mapstructure:"http_url_path" structs:"http_url_path" env:"OTEL_HTTP_URL_PATH"`
}

type Cos struct {
	SecretId  string `mapstructure:"secret_id" structs:"secret_id" env:"COS_SECRET_ID"`
	SecretKey string `mapstructure:"secret_key" structs:"secret_key" env:"COS_SECRET_KEY"`
	Bucket    string `mapstructure:"bucket" structs:"bucket" env:"COS_BUCKET"`
	Region    string `mapstructure:"region" structs:"region" env:"COS_REGION"`
}

func NewConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	// viper.AutomaticEnv()

	if err := bindEnv(&Config{}, ""); err != nil {
		return nil, fmt.Errorf("failed to bind environment variables: %s", err)
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to Read configuration: %s", err)
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to Unmarshal configuration: %s", err)
	}
	return cfg, nil
}

func bindEnv(data interface{}, prefix string) error {
	s := structs.New(data)
	for _, field := range s.Fields() {
		key := field.Tag("structs")
		env := field.Tag("env")
		if prefix != "" {
			key = fmt.Sprintf("%s.%s", prefix, key)
		}

		value := field.Value()
		if structs.IsStruct(value) {
			if err := bindEnv(value, key); err != nil {
				return err
			}
		} else {
			if env != "" {
				if err := viper.BindEnv(key, env); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
