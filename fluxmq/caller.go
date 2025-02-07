package fluxmq

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"

	wants "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
)

type Caller interface {
	Call(ctx context.Context, topic string, message *message.Message) (*message.Message, error)
	Close() error
}

type NatsCallerConfig struct {
	URL         string
	Marshaler   wants.Marshaler
	Unmarshaler wants.Unmarshaler
}

type NatsCaller struct {
	conn        *nats.Conn
	marshaler   wants.Marshaler
	unmarshaler wants.Unmarshaler
	// TODO: add logging via watermill.LoggerAdapter
}

func NewNatsCaller(
	cfg *NatsCallerConfig,
) (*NatsCaller, error) {
	if cfg == nil {
		cfg = new(NatsCallerConfig)
	}

	if cfg.URL == "" {
		cfg.URL = nats.DefaultURL
	}

	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	marshaller := new(wants.NATSMarshaler)

	if cfg.Marshaler == nil {
		cfg.Marshaler = marshaller
	}

	if cfg.Unmarshaler == nil {
		cfg.Unmarshaler = marshaller
	}

	return &NatsCaller{
		conn:        conn,
		marshaler:   cfg.Marshaler,
		unmarshaler: cfg.Unmarshaler,
	}, nil
}

func (nc *NatsCaller) Call(ctx context.Context, topic string, request *message.Message) (*message.Message, error) {
	natsRequest, err := nc.marshaler.Marshal(topic, request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nats message: %w", err)
	}

	natsResponse, err := nc.conn.RequestMsgWithContext(ctx, natsRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send nats message: %w", err)
	}

	response, err := nc.unmarshaler.Unmarshal(natsResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal nats message: %w", err)
	}

	return response, nil
}

func (nc *NatsCaller) Close() error {
	// TODO: should it close gracefully, like in watermill nats subscriber?

	nc.conn.Close()

	return nil
}
