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
	pub             message.Publisher
	sub             message.Subscriber
	call            fluxmq.Caller

	// onConnect func will be called in Run method after pub & sub creation
	onConnect func() error

	// onReady func will be called in Run method after getting config
	// it accepts raw payload from message.
	onReady func(r *message.Router, cfg NodesConfig[T]) error

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
		status:    NewAtomicValue(ServiceStatusStarting),
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

	router := options.routerFactory(options.watermillLogger)

	s.RegisterStatusHandler(router)

	err = s.run(ctx, options.routerFactory)
	if err != nil {
		return fmt.Errorf("failed to run service: %w", err)
	}

	return nil
}

func (s *Service[T]) run(ctx context.Context, routerFactory RouterFactory) error {
	configs, err := s.sub.Subscribe(ctx, s.topics.ResponseConfig())
	if err != nil {
		return fmt.Errorf("failed to subscribe to configs: %w", err)
	}

	var currentRouter *message.Router

	for msg := range configs {
		var config NodesConfig[T]
		err := json.Unmarshal(msg.Payload, &config)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		newRouter := routerFactory(s.watermillLogger)

		if s.onReady != nil {
			err := s.onReady(newRouter, config)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}
		}

		err = s.reloadNodes(ctx, &config, newRouter)
		if err != nil {
			return fmt.Errorf("failed to reload nodes: %w", err)
		}

		if currentRouter != nil {
			err := currentRouter.Close()
			if err != nil {
				return fmt.Errorf("failed to close router: %w", err)
			}
		}

		currentRouter = newRouter

		currentRouter.AddPlugin(func(_ *message.Router) error {
			return s.UpdateStatus(ServiceStatusReady)
		})

		msg.Ack()
		err = currentRouter.Run(ctx)
		if err != nil {
			return fmt.Errorf("failed to run router: %w", err)
		}
	}

	return nil
}

func (s *Service[T]) reloadNodes(ctx context.Context, nodes *NodesConfig[T], router *message.Router) error {
	for _, node := range s.nodes {
		if err := node.Close(); err != nil {
			s.logger.Error("failed to close node", slog.String("err", err.Error()))
		}
	}

	resultNodes := make([]Node[T], 0)
	for _, nodeCfg := range *nodes {
		node := NewNode[T](
			ctx,
			router,
			s.sub,
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
