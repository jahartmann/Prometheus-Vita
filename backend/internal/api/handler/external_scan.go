package handler

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// extractVMMAC returns the first MAC address from a Proxmox VM config map, where
// each netN entry looks like "<model>=<MAC>,bridge=vmbr0,...". Empty if none.
func extractVMMAC(cfg map[string]interface{}) string {
	for i := 0; i < 32; i++ {
		raw, ok := cfg[fmt.Sprintf("net%d", i)].(string)
		if !ok || raw == "" {
			continue
		}
		first := strings.SplitN(raw, ",", 2)[0]
		parts := strings.SplitN(first, "=", 2)
		if len(parts) == 2 {
			mac := strings.TrimSpace(parts[1])
			if strings.Count(mac, ":") == 5 {
				return strings.ToUpper(mac)
			}
		}
	}
	return ""
}

// parseNmapPorts parses `nmap -oG -` greppable output into open NodePorts.
// Line shape: "Host: <ip> ()\tPorts: 22/open/tcp//ssh//, 80/open/tcp//http//nginx/\t..."
func parseNmapPorts(greppable string) []NodePort {
	ports := []NodePort{}
	for _, line := range strings.Split(greppable, "\n") {
		idx := strings.Index(line, "Ports:")
		if idx < 0 {
			continue
		}
		section := line[idx+len("Ports:"):]
		if tab := strings.Index(section, "\t"); tab >= 0 {
			section = section[:tab]
		}
		for _, entry := range strings.Split(section, ",") {
			parts := strings.Split(strings.TrimSpace(entry), "/")
			if len(parts) < 3 {
				continue
			}
			portNum, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			if strings.TrimSpace(parts[1]) != "open" {
				continue
			}
			service := ""
			if len(parts) >= 5 {
				service = strings.TrimSpace(parts[4])
			}
			ports = append(ports, NodePort{
				Protocol:  strings.TrimSpace(parts[2]),
				State:     "LISTEN",
				LocalPort: portNum,
				Process:   service,
			})
		}
	}
	return ports
}

// scanQEMUPortsExternal discovers a QEMU VM's open ports WITHOUT the guest
// agent: it resolves the VM's IP from its MAC via the node's neighbor (ARP)
// table, then runs an nmap service scan from the node. This only sees
// network-reachable ports (not loopback-only services) but needs no in-guest
// agent. Returns the group unchanged (caller keeps the no_agent state) if the
// IP cannot be resolved or nmap yields nothing.
func (h *NodeHandler) scanQEMUPortsExternal(ctx context.Context, nodeID uuid.UUID, vmid int, group VMPortGroup) VMPortGroup {
	cfg, err := h.service.GetVMConfig(ctx, nodeID, vmid, "qemu")
	if err != nil {
		return group
	}
	mac := extractVMMAC(cfg)
	if mac == "" {
		return group
	}
	// Resolve the IP from the node's neighbor table (mac is hex+colons — safe to
	// interpolate; it passed the 5-colon check in extractVMMAC).
	ipRes, err := h.service.RunSSHCommand(ctx, nodeID,
		fmt.Sprintf("ip neigh show | grep -i '%s' | awk '{print $1}' | head -n1", mac))
	if err != nil || ipRes == nil {
		return group
	}
	ip := strings.TrimSpace(ipRes.Stdout)
	if net.ParseIP(ip) == nil {
		return group
	}
	// nmap service scan from the node (ip validated above).
	scanRes, err := h.service.RunSSHCommand(ctx, nodeID,
		fmt.Sprintf("nmap -sV -oG - %s 2>/dev/null", ip))
	if err != nil || scanRes == nil {
		return group
	}
	scanned := parseNmapPorts(scanRes.Stdout)
	if len(scanned) == 0 {
		return group
	}
	group.Ports = scanned
	group.ScanStatus = "external"
	group.ScanError = fmt.Sprintf("Externer Scan ohne Agent (%s) — nur erreichbare Ports", ip)
	return group
}
