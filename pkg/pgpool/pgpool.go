package pgpool

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"net"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Migrations        string        `env:"POSTGRES_MIGRATIONS"`
	PostgresHost      string        `env:"POSTGRES_HOST" env-default:"127.0.0.1"`
	PostgresPort      int           `env:"POSTGRES_PORT" env-default:"5432"`
	PostgresDB        string        `env:"POSTGRES_DB"   env-default:"postgres"`
	PostgresUser      string        `env:"POSTGRES_USER" env-default:"postgres"`
	PostgresPassword  string        `env:"POSTGRES_PASSWORD"`
	DisableTLS        bool          `env:"POSTGRES_DISABLE_TLS"`
	MinCons           int           `env:"POSTGRES_MIN_CONS" env-default:"5"`
	MaxCons           int           `env:"POSTGRES_MAX_CONS" env-default:"10"`
	MaxConnLifetime   time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME"    env-default:"1h"`
	MaxConnIdleTime   time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME"   env-default:"30m"`
	HealthCheckPeriod time.Duration `env:"POSTGRES_HEALTH_CHECK_PERIOD"  env-default:"1m"`
	ConnectTimeout    time.Duration `env:"POSTGRES_CONNECT_TIMEOUT"      env-default:"5s"`
}

func NewConfig() (*Config, error) {
	cfg := Config{}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) DSN() string {
	sslMode := "require"
	if c.DisableTLS {
		sslMode = "disable"
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:   net.JoinHostPort(c.PostgresHost, strconv.Itoa(c.PostgresPort)),
		Path:   c.PostgresDB,
	}

	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()

	return u.String()
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := NewConfig()
	if err != nil {
		return nil, err
	}

	pool, err := makePool(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to make pool: %w", err)
	}

	if cfg.Migrations == "" {
		return pool, nil
	}

	if err := runMigrations(cfg.Migrations, pool); err != nil {
		return nil, err
	}

	return pool, nil
}

func makePool(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}

	poolCfg.MinConns = int32(cfg.MinCons)
	poolCfg.MaxConns = int32(cfg.MaxCons)
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func runMigrations(migrations string, pool *pgxpool.Pool) error {
	source := fmt.Sprintf("file://%s", migrations)
	db := stdlib.OpenDBFromPool(pool)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	mig, err := migrate.NewWithDatabaseInstance(source, "postgresql", driver)
	if err != nil {
		return err
	}

	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
