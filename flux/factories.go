package flux

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
)

func DefaultPublisherFactory(url string) PublisherFactory {
	return func(logger watermill.LoggerAdapter) (message.Publisher, error) {
		logger = logger.With(watermill.LogFields{
			"url":       url,
			"component": "flux.publisher",
		})

		factory, err := nats.NewPublisher(
			nats.PublisherConfig{
				URL:               url,
				NatsOptions:       nil,
				Marshaler:         nil,
				SubjectCalculator: nil,
				JetStream: nats.JetStreamConfig{
					Disabled: true,
				},
			},
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create nats publisher: %w", err)
		}

		return factory, nil
	}
}

func DefaultSubscriberFactory(url string) SubscriberFactory {
	return func(logger watermill.LoggerAdapter) (message.Subscriber, error) {
		logger = logger.With(watermill.LogFields{
			"url":       url,
			"component": "flux.subscriber",
		})

		factory, err := nats.NewSubscriber(
			nats.SubscriberConfig{
				URL:               url,
				QueueGroupPrefix:  "",
				SubscribersCount:  0,
				CloseTimeout:      0,
				AckWaitTimeout:    0,
				SubscribeTimeout:  0,
				NatsOptions:       nil,
				Unmarshaler:       nil,
				SubjectCalculator: nil,
				NakDelay:          nil,
				JetStream: nats.JetStreamConfig{
					Disabled: true,
				},
			},
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create nats subscriber: %w", err)
		}

		return factory, nil
	}
}

func DefaultRouterFactory(logger watermill.LoggerAdapter) *message.Router {
	logger = logger.With(watermill.LogFields{
		"component": "flux.router",
	})

	return message.NewDefaultRouter(logger)
}
