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
	"fmt"
	"github.com/flux-agi/flux_go/flux"
)

func main() {
	ctx := context.TODO()

	service := flux.NewService[string]("keyFireTestService")

	service.OnNodeReady(func(cfg flux.NodeConfig[string]) error {
		fmt.Printf("node %s is ready\n", cfg.Alias)
		return nil
	})

	if _, err := service.Run(ctx); err != nil {
		return
	}
}
```

See nodes implementations at organisation repositories for more examples.
