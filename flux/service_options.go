package flux

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/flux-agi/flux_go/fluxmq"
)

type ServiceOptions struct {
	logger *slog.Logger
	pub    message.Publisher
	sub    message.Subscriber
	call   fluxmq.Caller
	state  *State
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

func WithServiceCall(call fluxmq.Caller) ServiceOption {
	return func(o *ServiceOptions) {
		o.call = call
	}
}

func WithServiceState(state *State) ServiceOption {
	return func(o *ServiceOptions) {
		o.state = state
	}
}
