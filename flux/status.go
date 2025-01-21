package flux

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Status string

const (
	StatusInitializing Status = "initializing"
	StatusConnected    Status = "CONNECTED"
	StatusReady        Status = "READY"
	StatusActive       Status = "ACTIVE"
	StatusPaused       Status = "PAUSED"
	StatusError        Status = "ERROR"
)

func (n *Service) RegisterStatusHandler(r *message.Router) {
	r.AddHandler(
		"flux.request_status",
		n.topics.RequestStatus(),
		n.sub,
		n.topics.SendStatus(),
		n.pub,
		n.handleStatusRequest,
	)
}

func (n *Service) handleStatusRequest(_ *message.Message) ([]*message.Message, error) {
	status := n.Status()

	return []*message.Message{
		message.NewMessage(watermill.NewUUID(), []byte(status)),
	}, nil
}

func (n *Service) UpdateStatus(status Status) error {
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
