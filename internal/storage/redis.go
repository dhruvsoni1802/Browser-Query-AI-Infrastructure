package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//This struct wraps the redis client with helper methods
type RedisClient struct {
	client *redis.Client
	ctx context.Context
}

//This function creates a new redis client
func NewRedisClient(addr string, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		Password: password,
		DB: db,

		//Connection Pool settings
		PoolSize:     10,
		MinIdleConns: 5,

		//Timeout settings
		DialTimeout: 5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping tests the connection
func (r *RedisClient) Ping() error {
	return r.client.Ping(r.ctx).Err()
}

