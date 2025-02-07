package flux

import "fmt"

type InputPort struct {
	Alias  string   `json:"alias"`
	Topics []string `json:"topics"`
}

type OutputPort struct {
	Alias string `json:"alias"`
}

func (o *OutputPort) Topic(nodeID string) string {
	return fmt.Sprintf("/node/%s/port/%s", nodeID, o.Alias)
}
