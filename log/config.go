package log

import (
	"time"
)

type Config struct {
	Debug   bool            `json:"debug" mapstructure:"debug"`
	Encoder string          `json:"encoder" mapstructure:"encoder"`
	Sampler []SamplerConfig `json:"sampler" mapstructure:"sampler,omitempty"`
}

type SamplerConfig struct {
	LevelRange LevelRangeConfig `json:"level_range" mapstructure:"level_range,omitempty"`
	Interval   time.Duration    `json:"interval" mapstructure:"interval"`
	First      int              `json:"first" mapstructure:"first"`
	Thereafter int              `json:"thereafter" mapstructure:"thereafter"`
}

type LevelRangeConfig struct {
	From LevelString `json:"from" mapstructure:"from,omitempty"`
	To   LevelString `json:"to" mapstructure:"to,omitempty"`
}
