# Go Flux SDK

In this repository you can find many utility functions
that will make it easier for you to develop services for Flux systems

## Node running

A node instance allows you to launch your application, notify Manager,
and request configuration from the editor.

We use watermill as abstraction for pub/sub, so you can use router to
register your own application handlers.

```go
package main

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flux-agi/flux_go/flux"
)

func main() {
	ctx := context.TODO()

	node := flux.NewNode("MyUniqueNodeName")
	defer node.Close(ctx)

	node.OnReady(InitRouter)

	router, err := node.Connect(ctx)
	if err != nil {
		panic(err)
	}

	err = router.Run(ctx)
	if err != nil {
		panic(err)
    }
}

func InitRouter(
	_ []byte,  // Config from editor
	router *message.Router,
	pub message.Publisher,
	sub message.Subscriber,
) error {
	router.AddHandler(
		"keyFireHandler",
		"input",
		sub,
		"output",
		pub,
		func(msg *message.Message) ([]*message.Message, error) {
			return []*message.Message{
				message.NewMessage(watermill.NewUUID(), []byte("OK")),
            }, nil
		},
	)

	return nil
}
```

## Subscribe to node port
```go
nodesCfg, _ := flux.GetConfig[string](ctx, service)
router := flux.DefaultRouterFactory(logger)

ports, _ := nodesCfg.NodesPortByAlias("port_name")
service.OnPort(ctx, ports, router, func(nodeAlias string, timestamp time.Time, payload []byte) {

})
```

See nodes implementations at organisation repositories for more examples.
