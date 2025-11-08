package log

import "errors"

var (
	// ErrSamplerLevelRangeInvalid indicates that a sampler level range is invalid.
	ErrSamplerLevelRangeInvalid = errors.New("sampler level range invalid")
	// ErrSamplerLevelRangeOverlap indicates that two sampler level ranges overlap.
	ErrSamplerLevelRangeOverlap = errors.New("sampler level range overlap")
)
