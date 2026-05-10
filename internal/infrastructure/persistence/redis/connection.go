package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

var Client *redis.Client

func Connect(cfg *config.RedisConfig) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	}

	if cfg.DialTimeout > 0 {
		opts.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = cfg.WriteTimeout
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	Client = client
	logger.Info("connected to Redis",
		zap.String("addr", cfg.Addr()),
		zap.Int("pool_size", cfg.PoolSize),
		zap.Int("min_idle_conns", cfg.MinIdleConns))

	return client, nil
}

func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

func Ping(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("redis not initialized")
	}
	return Client.Ping(ctx).Err()
}