package client

import (
	"os"

	"github.com/rs/zerolog"
)

const (
	DefaultTimeout = int64(30) // 30 sec
)

var DefaultLogger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()

type Option interface {
	Apply(*BlockchainClient)
}

type TimeoutOpt int64

func (t TimeoutOpt) Apply(c *BlockchainClient) {
	c.timeout = int64(t)
}
func WithTimeout(t int64) TimeoutOpt {
	if t <= 0 {
		panic("Timeout should be positive")
	}
	return TimeoutOpt(t)
}

type LoggerOpt zerolog.Logger

func (o LoggerOpt) Apply(c *BlockchainClient) {
	c.logger = zerolog.Logger(o)
}
func WithLoggerOpt(logger zerolog.Logger) LoggerOpt {
	return LoggerOpt(logger)
}
