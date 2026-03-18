package netscan

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

// SSHRunner is the interface used by all scanning functions in this package.
// The concrete *ssh.Client is adapted to this interface by poolClientRunner in
// scheduler.go, so the scanning logic remains free of direct SSH dependencies.
//
// Run executes a shell command on the remote node and returns its stdout,
// the process exit code, and any transport-level error.
type SSHRunner interface {
	Run(ctx context.Context, cmd string) (stdout string, exitCode int, err error)
}

// PortInfo holds parsed information about a single socket entry.
type PortInfo struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	State     string `json:"state"`
	Process   string `json:"process"`
	LocalAddr string `json:"local_addr"`
	PeerAddr  string `json:"peer_addr"`
	PeerPort  int    `json:"peer_port"`
}

// QuickScanResult is the output of RunQuickScan.
type QuickScanResult struct {
	ListeningTCP []PortInfo `json:"listening_tcp"`
	ListeningUDP []PortInfo `json:"listening_udp"`
	Established  []PortInfo `json:"established"`
}

// processRe matches the users:(("name",pid=N,...)) field in ss output.
var processRe = regexp.MustCompile(`users:\(\("([^"]+)"`)

// RunQuickScan executes ss commands over SSH and returns structured socket data.
// It runs three commands:
//   - ss -pltn  → TCP listening ports
//   - ss -plun  → UDP listening ports
//   - ss -tn state established → established TCP connections
func RunQuickScan(ctx context.Context, runner SSHRunner, nodeID string) (*QuickScanResult, error) {
	result := &QuickScanResult{}

	tcpOut, _, err := runner.Run(ctx, "ss -pltn")
	if err != nil {
		return nil, fmt.Errorf("netscan: ss -pltn on node %s: %w", nodeID, err)
	}
	result.ListeningTCP = parseSSOuput(tcpOut, "tcp", "LISTEN")

	udpOut, _, err := runner.Run(ctx, "ss -plun")
	if err != nil {
		return nil, fmt.Errorf("netscan: ss -plun on node %s: %w", nodeID, err)
	}
	result.ListeningUDP = parseSSOuput(udpOut, "udp", "UNCONN")

	estOut, _, err := runner.Run(ctx, "ss -tn state established")
	if err != nil {
		// Non-fatal: node may have no established connections.
		slog.Warn("netscan: ss established failed", slog.String("node_id", nodeID), slog.Any("error", err))
	} else {
		result.Established = parseSSOuput(estOut, "tcp", "ESTAB")
	}

	return result, nil
}

// parseSSOuput parses the textual output of an ss command.
//
// ss column layout (varies slightly by version but stable for -pltn/-plun):
//
//	State   Recv-Q  Send-Q  Local_Address:Port  Peer_Address:Port  Process
//
// For "ss -tn state established" there is no State column — the first column
// is Recv-Q. We detect this by checking whether the first non-header token
// looks like a number.
func parseSSOuput(output, protocol, defaultState string) []PortInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 1 {
		return nil
	}

	var ports []PortInfo
	for _, line := range lines[1:] { // skip header
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// Need at least Local and Peer address columns.
		// With State column: State Recv-Q Send-Q Local Peer [Process]
		// Without State col: Recv-Q Send-Q Local Peer [Process]
		var localField, peerField, processField string
		if len(fields) >= 5 {
			// Check if first field looks like a queue size (digit) → no State col
			if _, err := strconv.Atoi(fields[0]); err == nil {
				// No state column
				localField = fields[2]
				peerField = fields[3]
				if len(fields) > 4 {
					processField = fields[4]
				}
			} else {
				// Has state column
				localField = fields[3]
				peerField = fields[4]
				if len(fields) > 5 {
					processField = fields[5]
				}
			}
		} else {
			continue
		}

		localPort := extractPort(localField)
		peerAddr, peerPort := splitAddrPort(peerField)
		process := extractProcess(processField)

		ports = append(ports, PortInfo{
			Port:      localPort,
			Protocol:  protocol,
			State:     defaultState,
			Process:   process,
			LocalAddr: localField,
			PeerAddr:  peerAddr,
			PeerPort:  peerPort,
		})
	}
	return ports
}

// extractPort parses the port from an address:port string.
// Handles IPv4 (1.2.3.4:80), IPv6 ([::1]:80), and wildcard (*:80).
func extractPort(addrPort string) int {
	idx := strings.LastIndex(addrPort, ":")
	if idx < 0 {
		return 0
	}
	portStr := addrPort[idx+1:]
	n, _ := strconv.Atoi(portStr)
	return n
}

// splitAddrPort splits addr:port into (addr, port).
func splitAddrPort(addrPort string) (string, int) {
	idx := strings.LastIndex(addrPort, ":")
	if idx < 0 {
		return addrPort, 0
	}
	addr := addrPort[:idx]
	port, _ := strconv.Atoi(addrPort[idx+1:])
	return addr, port
}

// extractProcess extracts the process name from the ss users field.
// Format: users:(("name",pid=N,fd=M))
func extractProcess(field string) string {
	if m := processRe.FindStringSubmatch(field); len(m) == 2 {
		return m[1]
	}
	return ""
}
