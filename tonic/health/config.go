package health

type Config struct {
	StatusOK    int
	StatusNotOK int
}

var DefaultConfig = Config{
	StatusOK:    200,
	StatusNotOK: 503,
}

type ConfigOption func(*Config)

func WithStatusOK(code int) ConfigOption {
	return func(c *Config) {
		c.StatusOK = code
	}
}
func WithStatusNotOK(code int) ConfigOption {
	return func(c *Config) {
		c.StatusNotOK = code
	}
}
