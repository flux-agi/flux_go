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
	InputPorts  []string      `json:"input_ports,omitempty"`
	OutputPorts []string      `json:"output_ports,omitempty"`
	Timer       *TickSettings `json:"timer,omitempty"`
	Config      T             `json:"config,omitempty"`
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
