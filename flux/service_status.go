package flux

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type ServiceStatus string

const (
	ServiceStatusInitializing ServiceStatus = "initializing"
	ServiceStatusConnected    ServiceStatus = "CONNECTED"
	ServiceStatusReady        ServiceStatus = "READY"
	ServiceStatusActive       ServiceStatus = "ACTIVE"
	ServiceStatusPaused       ServiceStatus = "PAUSED"
	ServiceStatusError        ServiceStatus = "ERROR"
)

func (s *Service[T]) RegisterStatusHandler() {
	s.router.AddHandler(
		"flux.request_status",
		s.topics.RequestStatus(),
		s.sub,
		s.topics.SendStatus(),
		s.pub,
		s.handleStatusRequest,
	)
}

func (s *Service[T]) handleStatusRequest(_ *message.Message) ([]*message.Message, error) {
	status := s.Status()

	return []*message.Message{
		message.NewMessage(watermill.NewUUID(), []byte(status)),
	}, nil
}

func (s *Service[T]) UpdateStatus(status ServiceStatus) error {
	err := s.pub.Publish(
		s.topics.SendStatus(),
		message.NewMessage(watermill.NewUUID(), []byte(status)),
	)
	if err != nil {
		return fmt.Errorf("could not publish status message: %w", err)
	}

	s.status.Set(status)

	return nil
}
