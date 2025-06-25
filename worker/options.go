package worker

import "context"

type Config struct {
	ctx       context.Context
	queueSize int32
}

type Option func(c *Config)

func WithContext(ctx context.Context) Option {
	return func(c *Config) {
		c.ctx = ctx
	}
}

func WithQueueSize(size int32) Option {
	return func(c *Config) {
		c.queueSize = size
	}
}
