package flux_test

import (
	"context"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flux-agi/flux_go/flux"
	"github.com/flux-agi/flux_go/fluxtest"
)

func TestNode_GetConfig(t *testing.T) {
	t.Parallel()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	t.Cleanup(cancel)

	const (
		serviceName    = "keyFireTestService"
		internalConfig = "mySupperCoolConfig"
	)
	serviceConfig := []flux.NodeConfig[string]{
		{
			Alias:       serviceName,
			InputPorts:  nil,
			OutputPorts: nil,
			Config:      internalConfig,
		},
	}

	logger := watermill.NewStdLogger(true, true)

	pubSub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            0,
		Persistent:                     false,
		BlockPublishUntilSubscriberAck: false,
	}, logger)
	t.Cleanup(func() {
		err := pubSub.Close()
		assert.NoError(t, err)
	})

	pub, sub := pubSub, pubSub

	manager := fluxtest.NewFakeManager(pubSub, pubSub, map[string]any{
		serviceName: serviceConfig,
	})
	service := flux.NewService(serviceName, flux.WithServicePub(pub), flux.WithServiceSub(sub))
	t.Cleanup(func() { service.Close(ctx) })

	stop := manager.Run(ctx)
	t.Cleanup(stop)

	nodesCfg, err := flux.GetConfig[string](ctx, service)
	t.Logf("serviceConfig: %s", nodesCfg)

	require.NoError(t, err)
	assert.Equal(t, internalConfig, nodesCfg[0].Config)
}
