package netscan

import (
	"encoding/json"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
)

// portKey is used as a map key for deduplication.
type portKey struct {
	port     int
	protocol string
}

// connKey identifies a unique established connection.
type connKey struct {
	localPort int
	peerAddr  string
	peerPort  int
}

// ComputeDiff compares two QuickScanResults and returns the differences.
// It detects new listening ports, closed ports, service changes, and new
// established connections. "current" is the latest scan; "previous" is
// the scan to compare against.
func ComputeDiff(current, previous *QuickScanResult) model.ScanDiff {
	diff := model.ScanDiff{}

	if previous == nil {
		// No baseline to compare — everything is "new" but we don't flag it.
		return diff
	}

	// --- Listening TCP diff ---
	prevTCP := indexByPort(previous.ListeningTCP)
	currTCP := indexByPort(current.ListeningTCP)

	for k, curr := range currTCP {
		if prev, exists := prevTCP[k]; !exists {
			diff.NewPorts = append(diff.NewPorts, model.PortChange{
				DeviceIP:    "localhost",
				Port:        curr.Port,
				Protocol:    curr.Protocol,
				ServiceName: curr.Process,
			})
		} else if prev.Process != curr.Process && curr.Process != "" {
			diff.ServiceChanges = append(diff.ServiceChanges, model.ServiceChange{
				DeviceIP:   "localhost",
				Port:       curr.Port,
				OldService: prev.Process,
				NewService: curr.Process,
			})
		}
	}
	for k, prev := range prevTCP {
		if _, exists := currTCP[k]; !exists {
			diff.ClosedPorts = append(diff.ClosedPorts, model.PortChange{
				DeviceIP:    "localhost",
				Port:        prev.Port,
				Protocol:    prev.Protocol,
				ServiceName: prev.Process,
			})
		}
	}

	// --- Listening UDP diff ---
	prevUDP := indexByPort(previous.ListeningUDP)
	currUDP := indexByPort(current.ListeningUDP)

	for k, curr := range currUDP {
		if _, exists := prevUDP[k]; !exists {
			diff.NewPorts = append(diff.NewPorts, model.PortChange{
				DeviceIP:    "localhost",
				Port:        curr.Port,
				Protocol:    curr.Protocol,
				ServiceName: curr.Process,
			})
		}
	}
	for k, prev := range prevUDP {
		if _, exists := currUDP[k]; !exists {
			diff.ClosedPorts = append(diff.ClosedPorts, model.PortChange{
				DeviceIP:    "localhost",
				Port:        prev.Port,
				Protocol:    prev.Protocol,
				ServiceName: prev.Process,
			})
		}
	}

	// --- Established connections diff ---
	prevConns := indexByConn(previous.Established)
	for _, curr := range current.Established {
		k := connKey{curr.Port, curr.PeerAddr, curr.PeerPort}
		if _, exists := prevConns[k]; !exists {
			diff.NewConnections = append(diff.NewConnections, model.ConnectionChange{
				LocalPort: curr.Port,
				PeerIP:    curr.PeerAddr,
				PeerPort:  curr.PeerPort,
				Process:   curr.Process,
			})
		}
	}

	return diff
}

// indexByPort builds a map from portKey to PortInfo for O(1) lookups.
func indexByPort(ports []PortInfo) map[portKey]PortInfo {
	m := make(map[portKey]PortInfo, len(ports))
	for _, p := range ports {
		m[portKey{p.Port, p.Protocol}] = p
	}
	return m
}

// indexByConn builds a map from connKey to PortInfo for O(1) lookups.
func indexByConn(ports []PortInfo) map[connKey]PortInfo {
	m := make(map[connKey]PortInfo, len(ports))
	for _, p := range ports {
		m[connKey{p.Port, p.PeerAddr, p.PeerPort}] = p
	}
	return m
}

// Risk score weights.
const (
	weightNewPort        = 0.40 // new unknown open port is most suspicious
	weightServiceChange  = 0.25 // service on an existing port changed
	weightNewConnection  = 0.20 // new outbound established connection
	weightClosedPort     = 0.05 // disappearing port is low-risk (service stopped)
)

// ComputeRiskScore returns a [0.0, 1.0] risk score for a given diff.
// The score is a weighted sum capped at 1.0.
func ComputeRiskScore(diff model.ScanDiff) float64 {
	score := 0.0

	score += float64(len(diff.NewPorts)) * weightNewPort
	score += float64(len(diff.ServiceChanges)) * weightServiceChange
	score += float64(len(diff.NewConnections)) * weightNewConnection
	score += float64(len(diff.ClosedPorts)) * weightClosedPort

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// whitelist is the structure we expect inside whitelist_json.
// Each entry matches ports to ignore during diff reporting.
type whitelistEntry struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	// Service is optional; if set it must also match for the entry to be ignored.
	Service string `json:"service,omitempty"`
}

// ApplyWhitelist removes entries from the diff that match a whitelisted port/protocol.
// whitelist should be a JSON array of whitelistEntry objects. Malformed JSON is
// silently ignored (the diff is returned unchanged) and a descriptive error is
// written via the returned error value.
func ApplyWhitelist(diff *model.ScanDiff, whitelist json.RawMessage) error {
	if len(whitelist) == 0 {
		return nil
	}

	var entries []whitelistEntry
	if err := json.Unmarshal(whitelist, &entries); err != nil {
		return fmt.Errorf("netscan: parse whitelist: %w", err)
	}

	allowed := make(map[portKey]whitelistEntry, len(entries))
	for _, e := range entries {
		allowed[portKey{e.Port, e.Protocol}] = e
	}

	diff.NewPorts = filterPortChanges(diff.NewPorts, allowed)
	diff.ClosedPorts = filterPortChanges(diff.ClosedPorts, allowed)
	diff.ServiceChanges = filterServiceChanges(diff.ServiceChanges, allowed)

	return nil
}

func filterPortChanges(changes []model.PortChange, allowed map[portKey]whitelistEntry) []model.PortChange {
	result := changes[:0]
	for _, c := range changes {
		k := portKey{c.Port, c.Protocol}
		if entry, ok := allowed[k]; ok {
			// If whitelist entry specifies a service, only suppress when it matches.
			if entry.Service == "" || entry.Service == c.ServiceName {
				continue
			}
		}
		result = append(result, c)
	}
	return result
}

func filterServiceChanges(changes []model.ServiceChange, allowed map[portKey]whitelistEntry) []model.ServiceChange {
	result := changes[:0]
	for _, c := range changes {
		// Use "tcp" as default protocol for service change lookup since we don't
		// track protocol here; try both.
		if _, ok := allowed[portKey{c.Port, "tcp"}]; ok {
			continue
		}
		if _, ok := allowed[portKey{c.Port, "udp"}]; ok {
			continue
		}
		result = append(result, c)
	}
	return result
}
