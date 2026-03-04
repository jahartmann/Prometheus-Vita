package model

type TopologyNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`     // host, vm, ct, storage, network
	Label    string                 `json:"label"`
	Status   string                 `json:"status"`   // online, offline, running, stopped
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TopologyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

type TopologyGraph struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}
