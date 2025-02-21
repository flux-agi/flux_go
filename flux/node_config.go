package flux

import (
	"fmt"
	"strings"
)

// NodesConfig is a slice of NodeConfig
type NodesConfig[T any] []NodeConfig[T]

// NodeConfig is a node config
type NodeConfig[T any] struct {
	ID       string        `json:"id"`
	Inputs   []*Port       `json:"inputs"`
	Outputs  []*Port       `json:"outputs"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Timer    *TickSettings `json:"timer"`
	Settings T             `json:"settings"`
}

type Port struct {
	Alias  string   `json:"alias"`
	Topics []string `json:"topics"`
}

// TickSettings is a local tick settings of node.
type TickSettings struct {
	Type     TimerType `json:"type"`
	Interval int       `json:"intervalMs"` //nolint:tagliatelle
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
