package config

import (
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

type (
	Config struct {
		GRPC struct {
			Port        string `env:"GRPC_PORT" envDefault:"50051"`
			GatewayPort string `env:"GRPC_GATEWAY_PORT" envDefault:"8080"`
		}

		Clients struct {
			CallbackAddr    string        `env:"CALLBACK_ADDR" envDefault:""`
			CallbackTimeout time.Duration `env:"CALLBACK_TIMEOUT" envDefault:"5s"`
		}

		Kafka struct {
			Brokers          string        `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
			Topic            string        `env:"KAFKA_NOTIFICATIONS_TOPIC" envDefault:"order_status_notifications"`
			DLQTopic         string        `env:"KAFKA_DLQ_TOPIC" envDefault:"order_status_notifications_dlq"`
			ConsumerGroup    string        `env:"KAFKA_CONSUMER_GROUP" envDefault:"notifications"`
			BatchSize        int           `env:"KAFKA_BATCH_SIZE" envDefault:"10"`
			PollTimeoutMs    int           `env:"KAFKA_POLL_TIMEOUT_MS" envDefault:"1000"`
			SessionTimeoutMs int           `env:"KAFKA_SESSION_TIMEOUT_MS" envDefault:"10000"`
			RetryBackoff     time.Duration `env:"KAFKA_RETRY_BACKOFF" envDefault:"1s"`
			DLQTimeout       time.Duration `env:"KAFKA_DLQ_TIMEOUT" envDefault:"5s"`
		}

		App struct {
			ShutdownGracefulDelay time.Duration `env:"SHUTDOWN_GRACEFUL_DELAY" envDefault:"3s"`
		}
	}
)

func (c *Config) KafkaBrokerAddrs() []string {
	parts := strings.Split(c.Kafka.Brokers, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func New() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	return &cfg, err
}
