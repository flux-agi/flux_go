package flux

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

var ErrPredefinedPubSub = errors.New("pub and sub must be nil if you want to run app this way")

type Service[T any] struct {
	serviceName string
	logger      *slog.Logger

	watermillLogger watermill.LoggerAdapter
	pub             message.Publisher
	sub             message.Subscriber

	// onConnect func will be called in Run method after pub & sub creation
	onConnect func() error

	// onReady func will be called in Run method after getting config
	// it accepts raw payload from message.
	onReady func(cfg NodesConfig[T], r *message.Router, pub message.Publisher, sub message.Subscriber) error

	topics *ServiceTopics
	status *AtomicValue[ServiceStatus]
	state  *AtomicValue[[]byte]

	nodes        []Node[T]
	nodeHandlers NodeHandlers[T]
}

func NewService[T any](serviceName string, opts ...ServiceOption) *Service[T] {
	options := &ServiceOptions{
		logger: nil,
		pub:    nil,
		sub:    nil,
		state:  nil,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Service[T]{
		logger:      options.logger,
		pub:         options.pub,
		sub:         options.sub,
		serviceName: serviceName,
		onConnect:   nil,
		onReady:     nil,
		topics:      NewTopics(serviceName),
		status:      NewAtomicValue(ServiceStatusInitializing),
		state:       NewAtomicValue(options.state),
		nodes:       make([]Node[T], 0),
	}
}

//nolint:cyclop
func (n *Service[T]) Run(ctx context.Context, opts ...ConnectOption) (*message.Router, error) {
	if n.pub != nil || n.sub != nil {
		return nil, ErrPredefinedPubSub
	}

	options := &RunOptions{
		watermillLogger: watermill.NopLogger{},
		pubFactory:      DefaultPublisherFactory(DefaultNatsURL),
		subFactory:      DefaultSubscriberFactory(DefaultNatsURL),
		routerFactory:   DefaultRouterFactory,
		configTimeout:   DefaultConfigWaitingTimeout,
	}
	for _, opt := range opts {
		opt(options)
	}

	n.watermillLogger = options.watermillLogger

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

	err = n.UpdateStatus(ServiceStatusConnected)
	if err != nil {
		return nil, fmt.Errorf("failed to update service status: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, options.configTimeout)
	defer cancel()

	cfg, err := n.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if err := n.reloadNodes(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to reload nodes: %w", err)
	}

	r := options.routerFactory(options.watermillLogger)

	err = n.onReady(*cfg, r, n.pub, n.sub)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	err = n.UpdateStatus(ServiceStatusReady)
	if err != nil {
		return nil, fmt.Errorf("failed to update service status: %w", err)
	}

	n.RegisterStatusHandler(r)
	n.RegisterStateHandler(r)

	return r, nil
}

func (n *Service[T]) Close(ctx context.Context) {
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

func (n *Service[T]) Status() ServiceStatus {
	status, ok := n.status.Get()
	if !ok {
		return ServiceStatusPaused
	}

	return status
}

func (n *Service[T]) State() []byte {
	value, ok := n.state.Get()
	if !ok {
		return nil
	}

	return value
}

func (n *Service[T]) Pub() message.Publisher { return n.pub }

func (n *Service[T]) Sub() message.Subscriber { return n.sub }

func (n *Service[T]) reloadNodes(ctx context.Context, nodes *NodesConfig[T]) error {
	for _, node := range n.nodes {
		if err := node.Close(); err != nil {
			n.logger.Error("failed to close node", slog.String("err", err.Error()))
		}
	}

	resultNodes := make([]Node[T], 0)
	for _, nodeCfg := range *nodes {
		node := NewNode[T](
			ctx,
			n.watermillLogger,
			n.sub,
			n.pub,
			nodeCfg,
		)
		resultNodes = append(resultNodes, *node)
	}

	n.nodes = resultNodes

	return nil
}
