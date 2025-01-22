package flux

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
)

type ServiceOptions struct {
	logger *slog.Logger
	pub    message.Publisher
	sub    message.Subscriber
	state  []byte
}

type ServiceOption func(*ServiceOptions)

func WithServiceLogger(logger *slog.Logger) ServiceOption {
	return func(o *ServiceOptions) {
		o.logger = logger
	}
}

func WithServicePub(pub message.Publisher) ServiceOption {
	return func(o *ServiceOptions) {
		o.pub = pub
	}
}

func WithServiceSub(sub message.Subscriber) ServiceOption {
	return func(o *ServiceOptions) {
		o.sub = sub
	}
}

func WithServiceState(state []byte) ServiceOption {
	return func(o *ServiceOptions) {
		o.state = state
	}
}
