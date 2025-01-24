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

func (n *Service[T]) RegisterStatusHandler(r *message.Router) {
	r.AddHandler(
		"flux.request_status",
		n.topics.RequestStatus(),
		n.sub,
		n.topics.SendStatus(),
		n.pub,
		n.handleStatusRequest,
	)
}

func (n *Service[T]) handleStatusRequest(_ *message.Message) ([]*message.Message, error) {
	status := n.Status()

	return []*message.Message{
		message.NewMessage(watermill.NewUUID(), []byte(status)),
	}, nil
}

func (n *Service[T]) UpdateStatus(status ServiceStatus) error {
	err := n.pub.Publish(
		n.topics.SendStatus(),
		message.NewMessage(watermill.NewUUID(), []byte(status)),
	)
	if err != nil {
		return fmt.Errorf("could not publish status message: %w", err)
	}

	n.status.Set(status)

	return nil
}
