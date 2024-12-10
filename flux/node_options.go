package flux

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
)

type NodeOptions struct {
	logger *slog.Logger
	pub    message.Publisher
	sub    message.Subscriber
	state  []byte
}

type NodeOption func(*NodeOptions)

func WithNodeLogger(logger *slog.Logger) NodeOption {
	return func(o *NodeOptions) {
		o.logger = logger
	}
}

func WithNodePub(pub message.Publisher) NodeOption {
	return func(o *NodeOptions) {
		o.pub = pub
	}
}

func WithNodeSub(sub message.Subscriber) NodeOption {
	return func(o *NodeOptions) {
		o.sub = sub
	}
}

func WithNodeState(state []byte) NodeOption {
	return func(o *NodeOptions) {
		o.state = state
	}
}
