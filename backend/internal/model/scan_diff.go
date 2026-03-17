package model

type ScanDiff struct {
	NewPorts           []PortChange       `json:"new_ports"`
	ClosedPorts        []PortChange       `json:"closed_ports"`
	NewDevices         []DeviceChange     `json:"new_devices"`
	DisappearedDevices []DeviceChange     `json:"disappeared_devices"`
	ServiceChanges     []ServiceChange    `json:"service_changes"`
	NewConnections     []ConnectionChange `json:"new_connections"`
}

type PortChange struct {
	DeviceIP    string `json:"device_ip"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ServiceName string `json:"service_name,omitempty"`
}

type DeviceChange struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

type ServiceChange struct {
	DeviceIP   string `json:"device_ip"`
	Port       int    `json:"port"`
	OldService string `json:"old_service"`
	NewService string `json:"new_service"`
}

type ConnectionChange struct {
	LocalPort int    `json:"local_port"`
	PeerIP    string `json:"peer_ip"`
	PeerPort  int    `json:"peer_port"`
	Process   string `json:"process,omitempty"`
}
