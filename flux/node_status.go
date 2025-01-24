package flux

type NodeStatus string

const (
	NodeStatusReady  NodeStatus = "READY"
	NodeStatusActive NodeStatus = "RUNNING"
	NodeStatusPaused NodeStatus = "STOPPED"
	NodeStatusError  NodeStatus = "ERROR"
)
