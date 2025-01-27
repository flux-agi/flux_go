package flux

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	DefaultNatsURL              = "nats://localhost:4222"
	DefaultConfigWaitingTimeout = 5 * time.Second
)

type RunOptions struct {
	watermillLogger watermill.LoggerAdapter
	pubFactory      PublisherFactory
	subFactory      SubscriberFactory
	routerFactory   RouterFactory
	configTimeout   time.Duration
}

type ConnectOption func(*RunOptions)

type (
	PublisherFactory  = func(watermill.LoggerAdapter) (message.Publisher, error)
	SubscriberFactory = func(watermill.LoggerAdapter) (message.Subscriber, error)
	RouterFactory     = func(watermill.LoggerAdapter) *message.Router
)

func WithWatermillLogger(logger watermill.LoggerAdapter) ConnectOption {
	return func(o *RunOptions) {
		o.watermillLogger = logger
	}
}

func WithPublisherFactory(pf PublisherFactory) ConnectOption {
	return func(n *RunOptions) {
		n.pubFactory = pf
	}
}

func WithPublisher(pub message.Publisher) ConnectOption {
	return func(n *RunOptions) {
		n.pubFactory = func(_ watermill.LoggerAdapter) (message.Publisher, error) {
			return pub, nil
		}
	}
}

func WithSubscriberFactory(sf SubscriberFactory) ConnectOption {
	return func(n *RunOptions) {
		n.subFactory = sf
	}
}

func WithSubscriber(sub message.Subscriber) ConnectOption {
	return func(n *RunOptions) {
		n.subFactory = func(_ watermill.LoggerAdapter) (message.Subscriber, error) {
			return sub, nil
		}
	}
}

func WithRouterFactory(rf RouterFactory) ConnectOption {
	return func(n *RunOptions) {
		n.routerFactory = rf
	}
}

func WithConfigTimeout(configTimeout time.Duration) ConnectOption {
	return func(n *RunOptions) {
		n.configTimeout = configTimeout
	}
}
