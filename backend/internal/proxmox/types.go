package proxmox

type NodeStatus struct {
	Node        string    `json:"node"`
	Status      string    `json:"status"`
	Uptime      int64     `json:"uptime"`
	CPUUsage    float64   `json:"cpu_usage"`
	CPUCores    int       `json:"cpu_cores"`
	CPUModel    string    `json:"cpu_model"`
	MemTotal    int64     `json:"memory_total"`
	MemUsed     int64     `json:"memory_used"`
	MemFree     int64     `json:"memory_free"`
	SwapTotal   int64     `json:"swap_total"`
	SwapUsed    int64     `json:"swap_used"`
	DiskTotal   int64     `json:"disk_total"`
	DiskUsed    int64     `json:"disk_used"`
	NetIn       int64     `json:"net_in"`
	NetOut      int64     `json:"net_out"`
	LoadAvg     []float64 `json:"load_average"`
	KVersion    string    `json:"kernel_version"`
	PVEVersion  string    `json:"pve_version"`
	VMCount     int       `json:"vm_count"`
	VMRunning   int       `json:"vm_running"`
	CTCount     int       `json:"ct_count"`
	CTRunning   int       `json:"ct_running"`
}

type VMInfo struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Type      string  `json:"type"` // qemu or lxc
	CPU       float64 `json:"cpu"`
	CPUs      int     `json:"cpus"`
	MaxMem    int64   `json:"maxmem"`
	Mem       int64   `json:"mem"`
	MaxDisk   int64   `json:"maxdisk"`
	Disk      int64   `json:"disk"`
	Uptime    int64   `json:"uptime"`
	NetIn     float64 `json:"netin"`
	NetOut    float64 `json:"netout"`
	DiskRead  float64 `json:"diskread"`
	DiskWrite float64 `json:"diskwrite"`
	Tags      string  `json:"tags"`
}

type VMResponse struct {
	VMID        int     `json:"vmid"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	CPUUsage    float64 `json:"cpu_usage"`
	CPUCores    int     `json:"cpu_cores"`
	MemoryTotal int64   `json:"memory_total"`
	MemoryUsed  int64   `json:"memory_used"`
	DiskTotal   int64   `json:"disk_total"`
	DiskUsed    int64   `json:"disk_used"`
	Uptime      int64   `json:"uptime"`
	NetIn       int64   `json:"net_in"`
	NetOut      int64   `json:"net_out"`
	DiskRead    int64   `json:"disk_read"`
	DiskWrite   int64   `json:"disk_write"`
	Tags        string  `json:"tags"`
}

func (v VMInfo) ToResponse() VMResponse {
	return VMResponse{
		VMID:        v.VMID,
		Name:        v.Name,
		Type:        v.Type,
		Status:      v.Status,
		CPUUsage:    v.CPU * 100,
		CPUCores:    v.CPUs,
		MemoryTotal: v.MaxMem,
		MemoryUsed:  v.Mem,
		DiskTotal:   v.MaxDisk,
		DiskUsed:    v.Disk,
		Uptime:      v.Uptime,
		NetIn:       int64(v.NetIn),
		NetOut:      int64(v.NetOut),
		DiskRead:    int64(v.DiskRead),
		DiskWrite:   int64(v.DiskWrite),
		Tags:        v.Tags,
	}
}

// GuestOSInfo contains OS information from the QEMU guest agent.
type GuestOSInfo struct {
	ID             string `json:"id"`              // e.g. "debian", "ubuntu", "mswindows"
	Name           string `json:"name"`            // e.g. "Debian GNU/Linux", "Microsoft Windows 10"
	KernelRelease  string `json:"kernel-release"`
	KernelVersion  string `json:"kernel-version"`
	Machine        string `json:"machine"`
	PrettyName     string `json:"pretty-name"`
	Version        string `json:"version"`
	VersionID      string `json:"version-id"`
}

// OSFamily returns "windows" or "linux" based on the guest OS info.
func (o GuestOSInfo) OSFamily() string {
	if o.ID == "mswindows" {
		return "windows"
	}
	return "linux"
}

type StorageInfo struct {
	Storage      string  `json:"storage"`
	Type         string  `json:"type"`
	Content      string  `json:"content"`
	Total        int64   `json:"total"`
	Used         int64   `json:"used"`
	Available    int64   `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
	Active       bool    `json:"active"`
	Shared       bool    `json:"shared"`
}

type VersionInfo struct {
	Version string `json:"version"`
	Release string `json:"release"`
}

type NetworkInterface struct {
	Iface       string `json:"iface"`
	Type        string `json:"type"`
	CIDR        string `json:"cidr,omitempty"`
	Address     string `json:"address,omitempty"`
	Netmask     string `json:"netmask,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	Active      int    `json:"active"`
	Method      string `json:"method,omitempty"`
	Comments    string `json:"comments,omitempty"`
	BridgePorts string `json:"bridge_ports,omitempty"`
	Autostart   int    `json:"autostart"`
}

type DiskInfo struct {
	DevPath string `json:"devpath"`
	Size    int64  `json:"size"`
	Model   string `json:"model,omitempty"`
	Serial  string `json:"serial,omitempty"`
	Type    string `json:"type"`
	Health  string `json:"health,omitempty"`
	Wearout string `json:"wearout,omitempty"`
	GPT     int    `json:"gpt"`
	Vendor  string `json:"vendor,omitempty"`
}

// Task types for async Proxmox operations

type TaskStatus struct {
	UPID       string `json:"upid"`
	Node       string `json:"node"`
	Status     string `json:"status"` // "running", "stopped"
	ExitStatus string `json:"exitstatus"`
	Type       string `json:"type"`
	PID        int    `json:"pid"`
	StartTime  int64  `json:"starttime"`
	EndTime    int64  `json:"endtime,omitempty"`
}

func (t *TaskStatus) IsRunning() bool {
	return t.Status == "running"
}

func (t *TaskStatus) IsSuccess() bool {
	return t.Status == "stopped" && t.ExitStatus == "OK"
}

type TaskLogEntry struct {
	LineNum int    `json:"n"`
	Text    string `json:"t"`
}

type VzdumpOptions struct {
	Storage  string `json:"storage,omitempty"`
	Mode     string `json:"mode,omitempty"` // stop, snapshot, suspend
	Compress string `json:"compress,omitempty"`
	Remove   int    `json:"remove,omitempty"`
}

type VMRRDDataPoint struct {
	Time   float64 `json:"time"`
	CPU    float64 `json:"cpu"`
	Mem    float64 `json:"mem"`
	MaxMem float64 `json:"maxmem"`
	Disk   float64 `json:"disk"`
	MaxDisk float64 `json:"maxdisk"`
	NetIn  float64 `json:"netin"`
	NetOut float64 `json:"netout"`
}

type RRDDataPoint struct {
	Time      int64   `json:"time"`
	CPU       float64 `json:"cpu"`
	NetIn     float64 `json:"net_in"`
	NetOut    float64 `json:"net_out"`
	MemUsed   int64   `json:"mem_used"`
	MemTotal  int64   `json:"mem_total"`
	RootUsed  int64   `json:"root_used"`
	RootTotal int64   `json:"root_total"`
	LoadAvg   float64 `json:"load_avg"`
	IOWait    float64 `json:"io_wait"`
}

type SnapshotInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parent      string `json:"parent"`
	Snaptime    int64  `json:"snaptime"`
	VMState     int    `json:"vmstate"`
}

type StorageContent struct {
	Volid  string `json:"volid"`
	Format string `json:"format"`
	Size   int64  `json:"size"`
	CTime  int64  `json:"ctime"`
}

type VNCProxyResponse struct {
	Ticket string `json:"ticket"`
	Port   string `json:"port"`
	Cert   string `json:"cert"`
	User   string `json:"user"`
	UPID   string `json:"upid"`
}
