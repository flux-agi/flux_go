package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/flux-agi/flux_go/fluxmq"
)

type Service[T any] struct {
	serviceID string
	logger    *slog.Logger

	watermillLogger watermill.LoggerAdapter
	router          *message.Router
	pub             message.Publisher
	sub             message.Subscriber
	call            fluxmq.Caller

	// onConnect func will be called in Run method after pub & sub creation
	onConnect func() error

	// onReady func will be called in Run method after getting config
	// it accepts raw payload from message.
	onReady func(cfg NodesConfig[T]) error

	topics *ServiceTopics
	status *AtomicValue[ServiceStatus]
	state  *State

	nodes        []Node[T]
	nodeHandlers NodeHandlers[T]
}

func NewService[T any](opts ...ServiceOption) *Service[T] {
	serviceID, ok := os.LookupEnv("SERVICE_ID")
	if !ok {
		panic("SERVICE_ID is not set")
	}

	options := &ServiceOptions{
		logger: nil,
		pub:    nil,
		sub:    nil,
		call:   nil,
		state:  NewState(),
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Service[T]{
		logger:    options.logger,
		pub:       options.pub,
		sub:       options.sub,
		call:      options.call,
		serviceID: serviceID,
		onConnect: nil,
		onReady:   nil,
		topics:    NewTopics(serviceID),
		status:    NewAtomicValue(ServiceStatusInitializing),
		state:     options.state,
		nodes:     make([]Node[T], 0),
	}
}

//nolint:cyclop
func (s *Service[T]) Run(ctx context.Context, opts ...ConnectOption) error {
	if s.pub != nil || s.sub != nil {
		return fmt.Errorf("pub and sub must be nil if you want to run app this way")
	}

	options := &RunOptions{
		watermillLogger: watermill.NopLogger{},
		// TODO: use same connection for pub, sub and call.
		pubFactory:  DefaultPublisherFactory(DefaultNatsURL),
		subFactory:  DefaultSubscriberFactory(DefaultNatsURL),
		callFactory: DefaultCallerFactory(DefaultNatsURL),

		routerFactory: DefaultRouterFactory,
		configTimeout: DefaultConfigWaitingTimeout,
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

	s.router = options.routerFactory(options.watermillLogger)

	s.RegisterStatusHandler()

	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	resultNodes := make([]Node[T], 0, len(s.nodes))
	for _, nodeConfig := range *cfg {
		node := NewNode[T](
			ctx,
			s.router,
			s.sub,
			s.pub,
			nodeConfig,
		)

		if err := node.RegisterHandlers(&s.nodeHandlers); err != nil {
			return fmt.Errorf("failed to register node handlers: %w", err)
		}

		resultNodes = append(resultNodes, *node)
	}

	s.nodes = resultNodes

	if s.onReady != nil {
		if err := s.onReady(*cfg); err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	if err := s.UpdateStatus(ServiceStatusReady); err != nil {
		return fmt.Errorf("failed to update service status: %w", err)
	}

	if err := s.router.Run(ctx); err != nil {
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

func (s *Service[T]) State() *State {
	return s.state
}

func (s *Service[T]) Pub() message.Publisher { return s.pub }

func (s *Service[T]) Sub() message.Subscriber { return s.sub }

func (s *Service[T]) Router() *message.Router { return s.router }

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
