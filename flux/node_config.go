package flux

import (
	"fmt"
	"strings"
)

// NodesConfig is a slice of NodeConfig
type NodesConfig[T any] []NodeConfig[T]

// NodeConfig is a node config
type NodeConfig[T any] struct {
	ID      string        `json:"id,omitempty"`
	Inputs  []*Port       `json:"inputs,omitempty"`
	Outputs []*Port       `json:"outputs,omitempty"`
	Timer   *TickSettings `json:"timer,omitempty"`
	Config  T             `json:"config,omitempty"`
}

type Port struct {
	Alias  string   `json:"alias"`
	Topics []string `json:"topics"`
}

// TickSettings is a local tick settings of node.
type TickSettings struct {
	Type     TimerType `json:"type,omitempty"`
	Interval int       `json:"intervalMs,omitempty"` //nolint:tagliatelle
}

type TimerType string

const (
	TimerTypeGlobal TimerType = "GLOBAL"
	TimerTypeLocal  TimerType = "LOCAL"
	TimerTypeNone   TimerType = "NONE"
)

// GetTickSettingsByAlias returns tick settings by alias
func (n *NodesConfig[T]) GetTickSettingsByAlias() map[string]TickSettings {
	settings := map[string]TickSettings{}
	for _, n := range *n {
		settings[n.ID] = *n.Timer
	}
	return settings
}

// GetNodeByAlias returns node by alias
func (n *NodesConfig[T]) GetNodeByAlias(alias string) (*NodeConfig[T], error) {
	for _, node := range *n {
		if strings.EqualFold(node.ID, alias) {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("node with alias %s not found", alias)
}
