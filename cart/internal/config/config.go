package config

import (
	"net"
	"net/url"
	"time"

	"github.com/caarlos0/env/v10"
)

type (
	Config struct {
		Clients struct {
			LOMSGrpcAddr string `env:"LOMS_GRPC_ADDR" envDefault:"localhost:50052"`
		}

		GRPC struct {
			Port        string `env:"GRPC_PORT" envDefault:"50051"`
			GatewayPort string `env:"GRPC_GATEWAY_PORT" envDefault:"8080"`
		}

		PG struct {
			Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
			Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
			DB       string `env:"POSTGRES_DB" envDefault:"loms"`
			User     string `env:"POSTGRES_USER" envDefault:"loms"`
			Password string `env:"POSTGRES_PASSWORD" envDefault:"12345"`
		}

		PGPool struct {
			MaxConns          int32         `env:"POSTGRES_MAX_CONNS" envDefault:"8"`
			MinConns          int32         `env:"POSTGRES_MIN_CONNS" envDefault:"1"`
			HealthCheckPeriod time.Duration `env:"POSTGRES_HEALTH_CHECK_PERIOD" envDefault:"30s"`
			MaxConnLifetime   time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"0s"`
			MaxConnIdleTime   time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME" envDefault:"5m"`
		}

		App struct {
			ShutdownGracefulDelay time.Duration `env:"SHUTDOWN_GRACEFUL_DELAY" envDefault:"3s"`
		}

		CORS struct {
			DefaultAllowedOrigin string `env:"DEFAULT_ALLOWED_ORIGIN" envDefault:"http://localhost:5173"`
			MaxAgeInSeconds      string `env:"ACCESS_CONTROL_MAX_AGE_SECONDS" envDefault:"86400"`
		}
	}
)

func (c *Config) ConstructPostgresURL() string {
	postgresURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.PG.User, c.PG.Password),
		Host:     net.JoinHostPort(c.PG.Host, c.PG.Port),
		Path:     c.PG.DB,
		RawQuery: "sslmode=disable",
	}

	return postgresURL.String()
}

func New() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	return &cfg, err
}
