package flux_test

import (
	"context"
	"os"
	"os/signal"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flux-agi/flux_go/flux"
	"github.com/flux-agi/flux_go/fluxtest"
)

func Test_IsDevelopmentMode(t *testing.T) {
	t.Parallel()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	t.Cleanup(cancel)

	const (
		serviceName = "keyFireTestService"
		config      = "mySupperCoolConfig"
	)

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
		serviceName: config,
	})
	node := flux.NewService(serviceName, flux.WithServicePub(pub), flux.WithServiceSub(sub))
	t.Cleanup(func() {
		node.Close(ctx)
	})

	stop := manager.Run(ctx)
	t.Cleanup(stop)

	cfg, err := flux.GetConfig[string](ctx, node)

	require.NoError(t, err)
	assert.Equal(t, config, cfg)

	// test
	if err := node.EnableDevMode(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("dev_mode.ini"); err != nil {
		t.Fatalf("dev_mode.ini was not created")
	}

	f, _ := os.ReadFile("dev_mode.ini")
	t.Logf("Generated UUID: %s\n", f)

	if err := os.Remove("dev_mode.ini"); err != nil {
		t.Fatal(err)
	}
}
