package config

import (
	"time"
)

// Config is config interface
type Config interface {
	Raw() []byte
	Unmarshal(v interface{}) error
	Get(key string) (interface{}, bool)
	GetString(key string) (string, bool)
	GetBool(key string) (bool, bool)
	GetInt(key string) (int64, bool)
	GetFloat(key string) (float64, bool)
	GetDuration(key string) (time.Duration, bool)
	GetTime(key string) (time.Time, bool)
}
