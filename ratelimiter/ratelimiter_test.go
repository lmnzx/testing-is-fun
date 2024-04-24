package ratelimiter_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/lmnzx/testing-is-fun/ratelimiter"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRateLimiter(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "valkey/valkey:7.2",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)

	endpoint, err := container.Endpoint(ctx, "")
	assert.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})

	limiter := ratelimiter.New(client, 3, time.Minute)

	ip := "137.70.0.1"

	t.Run("correct path", func(t *testing.T) {
		res, err := limiter.AddAndCheckIfExceeds(ctx, net.ParseIP(ip))
		assert.NoError(t, err)

		// Rate should not be exceeded
		assert.False(t, res.IsExceeded())

		// Check the key
		assert.Equal(t, client.Get(ctx, ip).Val(), "1")

		client.FlushAll(ctx)
	})

	t.Run("check expiry", func(t *testing.T) {
		client.Set(ctx, ip, "3", 0)

		res, err := limiter.AddAndCheckIfExceeds(ctx, net.ParseIP(ip))
		assert.NoError(t, err)

		// Rate should be exceeded
		assert.True(t, res.IsExceeded())

		// Check expire time is set
		assert.Greater(t, client.ExpireTime(ctx, ip).Val(), time.Duration(0))
	})
}
