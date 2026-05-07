package config

import (
	"errors"
	"os"
	"strings"
)

type Settings struct {
	DatabaseURL  string
	KafkaBrokers []string
	ManagerToken string
	AgentToken   string
	OTelEndpoint string
	DemoMode     bool
}

func Load() (Settings, error) {
	settings := Settings{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		KafkaBrokers: strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		ManagerToken: os.Getenv("MANAGER_API_TOKEN"),
		AgentToken:   os.Getenv("AGENT_API_TOKEN"),
		OTelEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		DemoMode:     os.Getenv("DEMO_MODE") == "true",
	}
	if settings.ManagerToken == "" || settings.AgentToken == "" || settings.ManagerToken == settings.AgentToken {
		return Settings{}, errors.New("distinct manager and agent tokens are required")
	}
	if settings.DatabaseURL == "" || settings.KafkaBrokers[0] == "" {
		return Settings{}, errors.New("DATABASE_URL and KAFKA_BROKERS are required")
	}
	return settings, nil
}
