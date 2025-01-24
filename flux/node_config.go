package flux

import (
	"fmt"
	"strings"
)

// NodesConfig is a slice of NodeConfig
type NodesConfig[T any] []NodeConfig[T]

// NodeConfig is a node config
type NodeConfig[T any] struct {
	Alias       string        `json:"alias,omitempty"`
	InputPorts  []NodePort    `json:"input_ports,omitempty"`
	OutputPorts []NodePort    `json:"output_ports,omitempty"`
	Timer       *TickSettings `json:"timer,omitempty"`
	Config      T             `json:"config,omitempty"`
}

// NodePort is a node output/input port for communication
type NodePort struct {
	Alias string `json:"alias,omitempty"`
	Topic string `json:"topic,omitempty"`
}

type TimerType string

const (
	TimerTypeGlobal TimerType = "GLOBAL"
	TimerTypeLocal  TimerType = "LOCAL"
	TimerTypeNone   TimerType = "NONE"
)

// TickSettings is a local tick settings of node
type TickSettings struct {
	Type     TimerType `json:"type,omitempty"`
	Interval int       `json:"intervalMs,omitempty"`
}

// NodesPort is a map of node ports by node alias
type NodesPort map[string]NodePort

// NodesPortByAlias returns node ports by alias
func (n *NodesConfig[T]) NodesPortByAlias(alias string) (NodesPort, error) {
	var ports NodesPort
	for _, node := range *n {
		for _, port := range node.InputPorts {
			if strings.EqualFold(port.Alias, alias) {
				ports[node.Alias] = port
			}
		}
	}

	if len(ports) == 0 {
		return nil, fmt.Errorf("port with alias %s not found", alias)
	}

	return ports, nil
}

// GetTickSettingsByAlias returns tick settings by alias
func (n *NodesConfig[T]) GetTickSettingsByAlias() map[string]TickSettings {
	settings := map[string]TickSettings{}
	for _, n := range *n {
		settings[n.Alias] = *n.Timer
	}
	return settings
}

// GetNodeByAlias returns node by alias
func (n *NodesConfig[T]) GetNodeByAlias(alias string) (*NodeConfig[T], error) {
	for _, node := range *n {
		if strings.EqualFold(node.Alias, alias) {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("node with alias %s not found", alias)
}

// InputPortByAlias returns input port by alias
func (c *NodeConfig[T]) InputPortByAlias(alias string) (string, error) {
	for _, port := range c.InputPorts {
		if strings.EqualFold(port.Alias, alias) {
			return port.Topic, nil
		}
	}
	return "", fmt.Errorf("input port with alias %s not found", alias)
}

// OutputPortByAlias returns output port by alias
func (c *NodeConfig[T]) OutputPortByAlias(alias string) (string, error) {
	for _, port := range c.OutputPorts {
		if strings.EqualFold(port.Alias, alias) {
			return port.Topic, nil
		}
	}
	return "", fmt.Errorf("input port with alias %s not found", alias)
}
