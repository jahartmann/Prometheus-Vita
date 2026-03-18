package netscan

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// FullScanResult holds the ports discovered by a full nmap service scan.
type FullScanResult struct {
	Ports []ScannedPort `json:"ports"`
}

// ScannedPort holds parsed data from a single nmap port line.
type ScannedPort struct {
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	State          string `json:"state"`
	ServiceName    string `json:"service_name"`
	ServiceVersion string `json:"service_version"`
}

// DiscoveredHost holds information about a host found during subnet discovery.
type DiscoveredHost struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	MAC      string `json:"mac"`
}

// CheckNmapAvailable returns true when nmap is present on the remote node.
func CheckNmapAvailable(ctx context.Context, runner SSHRunner) bool {
	_, exitCode, err := runner.Run(ctx, "which nmap")
	if err != nil {
		slog.Debug("netscan: nmap availability check failed", slog.Any("error", err))
		return false
	}
	return exitCode == 0
}

// RunFullScan performs an nmap service-version scan of the top N ports on
// localhost and returns the parsed results.
//
// Command: nmap -sV --top-ports <topPorts> -oG - localhost
func RunFullScan(ctx context.Context, runner SSHRunner, nodeID string, topPorts int) (*FullScanResult, error) {
	if topPorts <= 0 {
		topPorts = 100
	}
	cmd := fmt.Sprintf("nmap -sV --top-ports %d -oG - localhost", topPorts)
	out, exitCode, err := runner.Run(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("netscan: nmap full scan on node %s: %w", nodeID, err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("netscan: nmap exited %d on node %s", exitCode, nodeID)
	}

	result := &FullScanResult{
		Ports: parseNmapGreppable(out),
	}
	return result, nil
}

// RunSubnetDiscovery runs an nmap ping scan on the given subnet and returns
// all live hosts.
//
// Command: nmap -sn -oG - <subnet>
func RunSubnetDiscovery(ctx context.Context, runner SSHRunner, subnet string) ([]DiscoveredHost, error) {
	if err := ValidateCIDR(subnet); err != nil {
		return nil, fmt.Errorf("netscan: invalid subnet for discovery: %w", err)
	}

	cmd := fmt.Sprintf("nmap -sn -oG - %s", subnet)
	out, exitCode, err := runner.Run(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("netscan: nmap subnet discovery: %w", err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("netscan: nmap subnet discovery exited %d", exitCode)
	}

	return parseNmapHosts(out), nil
}

// parseNmapGreppable parses greppable (-oG) nmap output for port information.
//
// Relevant line format:
//
//	Host: 127.0.0.1 (localhost)	Ports: 22/open/tcp//ssh//OpenSSH 8.9p1/, 80/open/tcp//http//nginx 1.24/
func parseNmapGreppable(output string) []ScannedPort {
	var ports []ScannedPort
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Host:") {
			continue
		}

		portsIdx := strings.Index(line, "Ports:")
		if portsIdx < 0 {
			continue
		}

		portsSection := line[portsIdx+len("Ports:"):]
		// Trim any trailing tab-delimited sections (Ignored State:...)
		if tabIdx := strings.Index(portsSection, "\t"); tabIdx >= 0 {
			portsSection = portsSection[:tabIdx]
		}

		for _, entry := range strings.Split(portsSection, ",") {
			entry = strings.TrimSpace(entry)
			// Format: port/state/proto//service//version/
			parts := strings.Split(entry, "/")
			if len(parts) < 3 {
				continue
			}

			port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			state := strings.TrimSpace(parts[1])
			proto := strings.TrimSpace(parts[2])

			serviceName := ""
			serviceVersion := ""
			if len(parts) >= 5 {
				serviceName = strings.TrimSpace(parts[4])
			}
			if len(parts) >= 7 {
				serviceVersion = strings.TrimSpace(parts[6])
			}

			ports = append(ports, ScannedPort{
				Port:           port,
				Protocol:       proto,
				State:          state,
				ServiceName:    serviceName,
				ServiceVersion: serviceVersion,
			})
		}
	}
	return ports
}

// parseNmapHosts parses greppable nmap output for host discovery results.
//
// Relevant line format:
//
//	Host: 192.168.1.1 (router.local)	Status: Up
//
// MAC info appears on a separate comment line:
//
//	# Nmap scan report for ...  MAC Address: AA:BB:CC:DD:EE:FF (Vendor)
func parseNmapHosts(output string) []DiscoveredHost {
	var hosts []DiscoveredHost
	lines := strings.Split(output, "\n")

	macByIP := make(map[string]string)
	// First pass: collect MAC addresses from comment lines.
	for _, line := range lines {
		if strings.HasPrefix(line, "#") && strings.Contains(line, "MAC Address:") {
			// e.g.: # Nmap scan report for 192.168.1.1 (host)   MAC Address: AA:BB:... (Vendor)
			macIdx := strings.Index(line, "MAC Address:")
			if macIdx >= 0 {
				rest := strings.TrimSpace(line[macIdx+len("MAC Address:"):])
				macFields := strings.Fields(rest)
				if len(macFields) > 0 {
					mac := macFields[0]
					// Extract IP from the line — it appears before the MAC section.
					ipPart := line[:macIdx]
					ipFields := strings.Fields(ipPart)
					if len(ipFields) > 0 {
						ip := ipFields[len(ipFields)-1]
						// Strip surrounding parens from hostname if any
						ip = strings.TrimSuffix(strings.TrimPrefix(ip, "("), ")")
						macByIP[ip] = mac
					}
				}
			}
		}
	}

	// Second pass: collect live hosts.
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Host:") {
			continue
		}
		// Format: Host: <IP> (<hostname>)\tStatus: Up
		statusIdx := strings.Index(line, "Status:")
		if statusIdx < 0 {
			continue
		}
		status := strings.TrimSpace(line[statusIdx+len("Status:"):])
		if !strings.HasPrefix(status, "Up") {
			continue
		}

		hostPart := strings.TrimSpace(line[len("Host:"):statusIdx])
		if tabIdx := strings.Index(hostPart, "\t"); tabIdx >= 0 {
			hostPart = hostPart[:tabIdx]
		}
		hostPart = strings.TrimSpace(hostPart)

		ip := hostPart
		hostname := ""
		if parenIdx := strings.Index(hostPart, " ("); parenIdx >= 0 {
			ip = strings.TrimSpace(hostPart[:parenIdx])
			hostname = strings.Trim(hostPart[parenIdx:], " ()")
		}

		hosts = append(hosts, DiscoveredHost{
			IP:       ip,
			Hostname: hostname,
			MAC:      macByIP[ip],
		})
	}
	return hosts
}
