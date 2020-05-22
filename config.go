package socket

import "time"

type Config struct {
	WriteWait         time.Duration
	PongWait          time.Duration
	HeartbeatRate     time.Duration
	MaxMessageSize    int64
	MessageBufferSize int
}

func newConfig() *Config {
	return &Config{
		WriteWait:         time.Second * 10,
		PongWait:          time.Second * 60,
		HeartbeatRate:     time.Second * 54,
		MaxMessageSize:    512,
		MessageBufferSize: 256,
	}
}
