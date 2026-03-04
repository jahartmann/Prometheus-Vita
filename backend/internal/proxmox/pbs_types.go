package proxmox

type PBSDatastore struct {
	Name    string `json:"store"`
	Path    string `json:"path,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type PBSDatastoreStatus struct {
	Store        string  `json:"store"`
	Total        int64   `json:"total"`
	Used         int64   `json:"used"`
	Available    int64   `json:"avail"`
	UsagePercent float64 `json:"usage_percent,omitempty"`
	GCStatus     string  `json:"gc-status,omitempty"`
}

type PBSBackupJob struct {
	ID          string `json:"id"`
	Store       string `json:"store"`
	Schedule    string `json:"schedule,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Remote      string `json:"remote,omitempty"`
	RemoteStore string `json:"remote-store,omitempty"`
}
