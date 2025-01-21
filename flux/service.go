package flux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Service struct {
	logger *slog.Logger

	pub         message.Publisher
	sub         message.Subscriber
	serviceName string

	// onConnect func will be called in Connect method after pub & sub creation
	onConnect func() error

	// onReady func will be called in Connect method after getting config
	// it accepts raw payload from message.
	onReady func(cfg []byte, r *message.Router, pub message.Publisher, sub message.Subscriber) error

	topics *Topics
	status *AtomicValue[Status]
	state  *AtomicValue[[]byte]
}

func NewService(
	serviceName string,
	opts ...ServiceOption,
) *Service {
	options := &ServiceOptions{
		logger: nil,
		pub:    nil,
		sub:    nil,
		state:  nil,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Service{
		logger:      options.logger,
		pub:         options.pub,
		sub:         options.sub,
		serviceName: serviceName,
		onConnect:   nil,
		onReady:     nil,
		topics:      NewTopics(serviceName),
		status:      NewAtomicValue(StatusInitializing),
		state:       NewAtomicValue(options.state),
	}
}

var ErrPredefinedPubSub = errors.New("pub and sub must be nil if you want to run app this way")

//nolint:cyclop
func (n *Service) Connect(ctx context.Context, opts ...ConnectOption) (*message.Router, error) {
	if n.pub != nil || n.sub != nil {
		return nil, ErrPredefinedPubSub
	}

	options := &ConnectOptions{
		watermillLogger: watermill.NopLogger{},
		pubFactory:      DefaultPublisherFactory(DefaultNatsURL),
		subFactory:      DefaultSubscriberFactory(DefaultNatsURL),
		routerFactory:   DefaultRouterFactory,
		configTimeout:   DefaultConfigWaitingTimeout,
	}
	for _, opt := range opts {
		opt(options)
	}

	var err error

	n.sub, err = options.subFactory(options.watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create nats sub: %w", err)
	}

	n.pub, err = options.pubFactory(options.watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create nats pub: %w", err)
	}

	if n.onConnect != nil {
		err := n.onConnect()
		if err != nil {
			return nil, err
		}
	}

	err = n.UpdateStatus(StatusConnected)
	if err != nil {
		return nil, fmt.Errorf("failed to update service status: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, options.configTimeout)
	defer cancel()

	cfg, err := n.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	r := options.routerFactory(options.watermillLogger)

	err = n.onReady(cfg, r, n.pub, n.sub)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	err = n.UpdateStatus(StatusReady)
	if err != nil {
		return nil, fmt.Errorf("failed to update service status: %w", err)
	}

	n.RegisterStatusHandler(r)
	n.RegisterStateHandler(r)

	return r, nil
}

func (n *Service) Close(ctx context.Context) {
	if n.pub != nil {
		err := n.pub.Close()
		if err != nil {
			n.logger.ErrorContext(ctx, "failed to close publisher", slog.String("err", err.Error()))
		}

		n.pub = nil
	}

	if n.sub != nil {
		err := n.sub.Close()
		if err != nil {
			n.logger.ErrorContext(ctx, "failed to close subscriber", slog.String("err", err.Error()))
		}

		n.sub = nil
	}
}

func (n *Service) Status() Status {
	status, ok := n.status.Get()
	if !ok {
		return StatusPaused
	}

	return status
}

func (n *Service) State() []byte {
	value, ok := n.state.Get()
	if !ok {
		return nil
	}

	return value
}

func (n *Service) Pub() message.Publisher {
	return n.pub
}

// PubData sends message to topic
func (n *Service) PubData(topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload - %w", err)
	}
	return n.pub.Publish(topic, message.NewMessage(watermill.NewUUID(), data))
}

func (n *Service) Sub() message.Subscriber {
	return n.sub
}
