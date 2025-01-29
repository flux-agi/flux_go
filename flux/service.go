package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

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

func NewService[T any](opts ...ServiceOption) *Service[T] {
	serviceName, ok := os.LookupEnv("SERVICE_ALIAS")
	if !ok {
		panic(fmt.Errorf("SERVICE_ALIAS is not set"))
	}

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
func (s *Service[T]) Run(ctx context.Context, opts ...ConnectOption) error {
	if s.pub != nil || s.sub != nil {
		return fmt.Errorf("pub and sub must be nil if you want to run app this way")
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

	s.watermillLogger = options.watermillLogger

	var err error

	s.sub, err = options.subFactory(options.watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create nats sub: %w", err)
	}

	s.pub, err = options.pubFactory(options.watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create nats pub: %w", err)
	}

	if s.onConnect != nil {
		err := s.onConnect()
		if err != nil {
			return err
		}
	}

	err = s.UpdateStatus(ServiceStatusConnected)
	if err != nil {
		return fmt.Errorf("failed to update service status: %w", err)
	}

	r := options.routerFactory(options.watermillLogger)

	s.RegisterStatusHandler(r)
	s.RegisterStateHandler(r)
	s.RegisterConfigHandler(ctx, r)

	if err := r.Run(ctx); err != nil {
		return fmt.Errorf("failed to run router: %w", err)
	}

	return nil
}

func (s *Service[T]) Close(ctx context.Context) {
	if s.pub != nil {
		err := s.pub.Close()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to close publisher", slog.String("err", err.Error()))
		}

		s.pub = nil
	}

	if s.sub != nil {
		err := s.sub.Close()
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to close subscriber", slog.String("err", err.Error()))
		}

		s.sub = nil
	}
}

func (s *Service[T]) Status() ServiceStatus {
	status, ok := s.status.Get()
	if !ok {
		return ServiceStatusPaused
	}

	return status
}

func (s *Service[T]) State() []byte {
	value, ok := s.state.Get()
	if !ok {
		return nil
	}

	return value
}

func (s *Service[T]) Pub() message.Publisher { return s.pub }

func (s *Service[T]) PubToTopic(topic string, value any) error {
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	return s.pub.Publish(topic, message.NewMessage(watermill.NewUUID(), data))
}

func (s *Service[T]) PubToPort(port string, value any) error {
	for _, node := range s.nodes {
		if err := node.Push(port, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service[T]) Sub() message.Subscriber { return s.sub }

func (s *Service[T]) reloadNodes(ctx context.Context, nodes *NodesConfig[T]) error {
	for _, node := range s.nodes {
		if err := node.Close(); err != nil {
			s.logger.Error("failed to close node", slog.String("err", err.Error()))
		}
	}

	resultNodes := make([]Node[T], 0)
	for _, nodeCfg := range *nodes {
		router := DefaultRouterFactory(s.watermillLogger)
		sub, err := DefaultSubscriberFactory(DefaultNatsURL)(s.watermillLogger)
		if err != nil {
			return fmt.Errorf("failed to create nats sub: %w", err)
		}

		node := NewNode[T](
			ctx,
			router,
			sub,
			s.pub,
			nodeCfg,
		)

		if err := node.RegisterHandlers(&s.nodeHandlers); err != nil {
			return fmt.Errorf("failed to register node handlers: %w", err)
		}

		resultNodes = append(resultNodes, *node)
	}

	s.nodes = resultNodes

	return nil
}

func (s *Service[T]) RegisterConfigHandler(ctx context.Context, r *message.Router) {
	r.AddNoPublisherHandler(
		"flux.service.response_config",
		s.topics.ResponseConfig(),
		s.sub,
		func(msg *message.Message) error {
			var cfg NodesConfig[T]
			if err := json.Unmarshal(msg.Payload, &cfg); err != nil {
				return fmt.Errorf("failed to unmarshal config: %w", err)
			}

			if s.onReady != nil {
				if err := s.onReady(cfg, r, s.pub, s.sub); err != nil {
					return fmt.Errorf("failed to read config: %w", err)
				}
			}

			if err := s.reloadNodes(ctx, &cfg); err != nil {
				return fmt.Errorf("failed to reload nodes: %w", err)
			}

			if err := s.UpdateStatus(ServiceStatusReady); err != nil {
				return fmt.Errorf("failed to update service status: %w", err)
			}

			msg.Ack()

			return nil
		},
	)
}
