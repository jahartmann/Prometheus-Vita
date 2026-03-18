package logscan

// BuiltinSource describes a well-known Proxmox log file that should always be
// tracked on every node.
type BuiltinSource struct {
	Path       string
	ParserType string
}

// BuiltinSources lists the log files that are seeded automatically for every
// node during discovery.
var BuiltinSources = []BuiltinSource{
	{Path: "/var/log/syslog", ParserType: "syslog"},
	{Path: "/var/log/auth.log", ParserType: "syslog"},
	{Path: "/var/log/pveproxy/access.log", ParserType: "access_log"},
	{Path: "/var/log/pvedaemon.log", ParserType: "syslog"},
	{Path: "/var/log/pve-firewall.log", ParserType: "firewall"},
	{Path: "/var/log/corosync/corosync.log", ParserType: "corosync"},
	{Path: "/var/log/pve/tasks/active", ParserType: "proxmox_task"},
}
