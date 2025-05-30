package flux

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/flux-agi/flux_go/fluxmq"
)

type Service[T any] struct {
	serviceID string
	logger    *slog.Logger

	pub             message.Publisher
	sub             message.Subscriber
	call            fluxmq.Caller
	connectionMutex *sync.RWMutex

	// onConnect func will be called in Run method after pub & sub creation
	onConnect func() error

	// onReady func will be called in Run method after getting config
	// it accepts raw payload from message.
	onReady func(r *message.Router, cfg NodesConfig[T]) error

	// onIDEStatus is a function that will be called, when IDE status received.
	// manager send status to the service, when websocket connected/disconnected
	onIDEStatus func(status IDEStatusMessage) error

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
		logger:          options.logger,
		pub:             options.pub,
		sub:             options.sub,
		call:            options.call,
		connectionMutex: new(sync.RWMutex),
		serviceID:       serviceID,
		onConnect:       nil,
		onReady:         nil,
		onIDEStatus:     nil,
		topics:          NewTopics(serviceID),
		status:          NewAtomicValue(ServiceStatusStarting),
		state:           options.state,
		nodes:           make([]Node[T], 0),
	}
}

//nolint:cyclop
func (s *Service[T]) Run(ctx context.Context, opts ...ConnectOption) error {
	if s.pub != nil || s.sub != nil {
		return fmt.Errorf("pub and sub must be nil if you want to run app this way")
	}

	url := cmp.Or(os.Getenv("NATS_URL"), DefaultNatsURL)

	options := &RunOptions{
		watermillLogger: watermill.NopLogger{},
		// TODO: use same connection for pub, sub and call.
		pubFactory:  DefaultPublisherFactory(url),
		subFactory:  DefaultSubscriberFactory(url),
		callFactory: DefaultCallerFactory(url),

		routerFactory: DefaultRouterFactory,
		configTimeout: DefaultConfigWaitingTimeout,
	}
	for _, opt := range opts {
		opt(options)
	}

	err := s.connect(options)
	if err != nil {
		return fmt.Errorf("failed to connect service: %w", err)
	}

	if s.onConnect != nil {
		err := s.onConnect()
		if err != nil {
			return err
		}
	}

	err = s.UpdateStatus(ServiceStatusConnected)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	err = s.run(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to run service: %w", err)
	}

	return nil
}

//nolint:cyclop
func (s *Service[T]) run(ctx context.Context, options *RunOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	configSub, err := options.subFactory(options.watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create config subscriber: %w", err)
	}

	configs, err := configSub.Subscribe(ctx, s.topics.ResponseConfig())
	if err != nil {
		return fmt.Errorf("failed to subscribe to configs: %w", err)
	}

	err = s.requestConfig()
	if err != nil {
		return fmt.Errorf("failed to request config: %w", err)
	}

	var router *message.Router

	for msg := range configs {
		slog.DebugContext(ctx, "new confing was received")
		var config NodesConfig[T]
		err := json.Unmarshal(msg.Payload, &config)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		router, err = s.reinitRouter(router, options)
		if err != nil {
			return fmt.Errorf("failed to reinit router: %w", err)
		}

		if s.onReady != nil {
			err := s.onReady(router, config)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}
		}

		err = s.reloadNodes(ctx, &config, router)
		if err != nil {
			return fmt.Errorf("failed to reload nodes: %w", err)
		}

		msg.Ack()

		go func() {
			err = router.Run(ctx)
			if err != nil {
				slog.ErrorContext(
					ctx,
					"Failed to run router",
					slog.String("service", s.serviceID),
					slog.String("err", err.Error()),
				)
				cancel()
			}
		}()
	}

	return nil
}

func (s *Service[T]) reinitRouter(router *message.Router, options *RunOptions) (*message.Router, error) {
	var err error

	if router != nil {
		err = s.disconnect()
		if err != nil {
			return nil, fmt.Errorf("failed to disconnect: %w", err)
		}

		err = router.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to close router: %w", err)
		}

		err = s.connect(options)
		if err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	router, err = s.initRouter(options)
	if err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	return router, nil
}

func (s *Service[T]) connect(options *RunOptions) error {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	var err error

	s.sub, err = options.subFactory(options.watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create nats sub: %w", err)
	}

	s.pub, err = options.pubFactory(options.watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create nats pub: %w", err)
	}

	s.call, err = options.callFactory(options.watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create nats call: %w", err)
	}

	return nil
}

func (s *Service[T]) disconnect() error {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	if s.sub != nil {
		err := s.sub.Close()
		if err != nil {
			return fmt.Errorf("failed to close subscriber: %w", err)
		}
		s.sub = nil
	}

	if s.pub != nil {
		err := s.pub.Close()
		if err != nil {
			return fmt.Errorf("failed to close publisher: %w", err)
		}
		s.pub = nil
	}

	if s.call != nil {
		err := s.call.Close()
		if err != nil {
			return fmt.Errorf("failed to close call: %w", err)
		}
		s.call = nil
	}

	return nil
}

func (s *Service[T]) requestConfig() error {
	err := s.pub.Publish(s.topics.RequestConfig(), message.NewMessage(watermill.NewUUID(), []byte(s.serviceID)))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (s *Service[T]) initRouter(options *RunOptions) (*message.Router, error) {
	router := options.routerFactory(options.watermillLogger)
	s.RegisterStatusHandler(router)
	s.RegisterIDEStatusHandler(router)
	router.AddPlugin(func(_ *message.Router) error {
		return s.UpdateStatus(ServiceStatusReady)
	})

	return router, nil
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

func (s *Service[T]) Pub() message.Publisher {
	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()

	return s.pub
}

func (s *Service[T]) Sub() message.Subscriber {
	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()

	return s.sub
}

func (s *Service[T]) Call() fluxmq.Caller {
	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()

	return s.call
}

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
