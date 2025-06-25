package queue

type Config struct {
	Capacity int
}

type Option func(*Config)

func WithCapacity(cap int) Option {
	return func(c *Config) {
		if cap >= 0 {
			c.Capacity = cap
		}
	}
}
