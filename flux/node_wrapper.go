package flux

import (
	"fmt"
	"strings"
)

// NodesConfig is a slice of NodeConfig
type NodesConfig[T any] []NodeConfig[T]

// NodeConfig is a node config
type NodeConfig[T any] struct {
	Alias       string     `json:"alias,omitempty"`
	InputPorts  []NodePort `json:"input_ports,omitempty"`
	OutputPorts []NodePort `json:"output_ports,omitempty"`
	Config      T          `json:"config,omitempty"`
}

// NodePort is a node output/input port for communication
type NodePort struct {
	Alias string `json:"alias,omitempty"`
	Topic string `json:"topic,omitempty"`
}

// GetNodeByAlias returns node by alias
func (c *NodesConfig[T]) GetNodeByAlias(alias string) (*NodeConfig[T], error) {
	for _, node := range *c {
		if strings.EqualFold(node.Alias, alias) {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("node with alias %s not found", alias)
}

// GetInputByAlias returns input port by alias
func (c *NodeConfig[T]) GetInputByAlias(alias string) (string, error) {
	for _, port := range c.InputPorts {
		if strings.EqualFold(port.Alias, alias) {
			return port.Topic, nil
		}
	}
	return "", fmt.Errorf("input port with alias %s not found", alias)
}

// GetOutputByAlias returns output port by alias
func (c *NodeConfig[T]) GetOutputByAlias(alias string) (string, error) {
	for _, port := range c.OutputPorts {
		if strings.EqualFold(port.Alias, alias) {
			return port.Topic, nil
		}
	}
	return "", fmt.Errorf("input port with alias %s not found", alias)
}
