package netscan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BandwidthTestResult is what /nodes/:id/bandwidth-test returns to the
// frontend. All bit rates are bits per second; bytes are bytes total.
type BandwidthTestResult struct {
	SourceNodeID string  `json:"source_node_id"`
	TargetNodeID string  `json:"target_node_id"`
	TargetHost   string  `json:"target_host"`
	DurationSec  int     `json:"duration_sec"`
	Protocol     string  `json:"protocol"` // "tcp" or "udp"
	Direction    string  `json:"direction"` // "send" (source→target) or "reverse"

	BitsPerSecond float64 `json:"bits_per_second"`
	BytesTotal    int64   `json:"bytes_total"`
	Retransmits   int     `json:"retransmits,omitempty"`

	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`

	Warnings []string `json:"warnings,omitempty"`
	RawJSON  string   `json:"-"`
}

// BandwidthTestOptions controls how the test is executed.
type BandwidthTestOptions struct {
	DurationSec int    // 1..60, default 5
	Port        int    // default 5201
	Protocol    string // "tcp" (default) or "udp"
	Reverse     bool   // server sends to client
	TargetHost  string // explicit hostname/IP — defaults to the target node's hostname
}

// RunBandwidthTest executes an iperf3 test between two nodes via SSH.
//
// The flow:
//  1. Preflight: ensure iperf3 is installed on both source and target.
//  2. Start iperf3 in server mode on the target (background, single-shot).
//  3. Run iperf3 in client mode on the source against the target host.
//  4. Stop the server (best effort) and parse the client's JSON output.
//
// We use the JSON output mode (--json) so parsing is robust against
// version-specific text formats.
type bandwidthRunner interface {
	Run(ctx context.Context, cmd string) (stdout string, exitCode int, err error)
}

// RunBandwidthTest performs the iperf3 measurement and returns parsed results.
// `srcRunner` runs commands on the source node, `dstRunner` on the target.
// `srcID` and `dstID` are passed through into the result for the UI.
func RunBandwidthTest(
	ctx context.Context,
	srcRunner bandwidthRunner,
	dstRunner bandwidthRunner,
	srcID, dstID string,
	opts BandwidthTestOptions,
) (*BandwidthTestResult, error) {
	if opts.DurationSec <= 0 {
		opts.DurationSec = 5
	}
	if opts.DurationSec > 60 {
		opts.DurationSec = 60
	}
	if opts.Port == 0 {
		opts.Port = 5201
	}
	if opts.Protocol == "" {
		opts.Protocol = "tcp"
	}
	if opts.TargetHost == "" {
		return nil, fmt.Errorf("Ziel-Host fehlt — Bandbreitentest benötigt eine erreichbare IP/Hostname")
	}

	res := &BandwidthTestResult{
		SourceNodeID: srcID,
		TargetNodeID: dstID,
		TargetHost:   opts.TargetHost,
		DurationSec:  opts.DurationSec,
		Protocol:     opts.Protocol,
		Direction:    "send",
		StartedAt:    time.Now(),
	}
	if opts.Reverse {
		res.Direction = "reverse"
	}

	// 1) Preflight on both ends — fail fast if iperf3 is missing.
	if err := ensureIperf3(ctx, srcRunner, "Quell-Node"); err != nil {
		return nil, err
	}
	if err := ensureIperf3(ctx, dstRunner, "Ziel-Node"); err != nil {
		return nil, err
	}

	// 2) Start iperf3 server on target. -1 = single-connection then exit, so we
	//    do not need an explicit kill in the happy path. We run it in the
	//    background and add a kill-on-exit safety net.
	serverCmd := fmt.Sprintf(
		"(iperf3 --server --one-off --port %d --daemon >/dev/null 2>&1; true); sleep 1",
		opts.Port,
	)
	if _, _, err := dstRunner.Run(ctx, serverCmd); err != nil {
		return nil, fmt.Errorf("iperf3-Server konnte nicht starten: %w", err)
	}
	defer func() {
		// Best-effort cleanup — ignore errors. With --one-off the server should
		// already be gone, but we kill any leftover daemon just in case.
		killCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _, _ = dstRunner.Run(killCtx, fmt.Sprintf("pkill -f 'iperf3.*--port %d' 2>/dev/null || true", opts.Port))
	}()

	// 3) Run iperf3 client. --json for stable parsing, --connect-timeout to
	//    fail fast when networking is broken.
	args := []string{
		"iperf3",
		"--client", shellEscape(opts.TargetHost),
		"--port", fmt.Sprintf("%d", opts.Port),
		"--time", fmt.Sprintf("%d", opts.DurationSec),
		"--connect-timeout", "5000",
		"--json",
	}
	if opts.Protocol == "udp" {
		args = append(args, "--udp")
	}
	if opts.Reverse {
		args = append(args, "--reverse")
	}
	clientCmd := strings.Join(args, " ")

	cctx, cancel := context.WithTimeout(ctx, time.Duration(opts.DurationSec+15)*time.Second)
	defer cancel()
	stdout, exit, err := srcRunner.Run(cctx, clientCmd)
	res.CompletedAt = time.Now()
	if err != nil {
		return nil, fmt.Errorf("iperf3-Client fehlgeschlagen: %w (Output: %s)", err, truncate(stdout, 200))
	}
	res.RawJSON = stdout
	if exit != 0 {
		res.Warnings = append(res.Warnings, fmt.Sprintf("iperf3 endete mit Exit-Code %d", exit))
	}

	// 4) Parse JSON output — iperf3's structure is stable across recent versions.
	if err := parseIperf3JSON(stdout, res); err != nil {
		// We still return what we have, but mark it as a warning. The frontend
		// can show the raw output for debugging.
		res.Warnings = append(res.Warnings, fmt.Sprintf("Antwort konnte nicht vollständig geparst werden: %v", err))
	}
	return res, nil
}

func ensureIperf3(ctx context.Context, runner bandwidthRunner, who string) error {
	out, _, err := runner.Run(ctx, "command -v iperf3 || true")
	if err != nil {
		return fmt.Errorf("%s nicht erreichbar: %w", who, err)
	}
	if strings.TrimSpace(out) == "" {
		return fmt.Errorf("iperf3 ist auf %s nicht installiert. Installation: 'apt install iperf3' (Debian/Ubuntu) oder 'apk add iperf3' (Alpine).", who)
	}
	return nil
}

// parseIperf3JSON pulls the relevant numbers out of iperf3 --json output.
// We deliberately tolerate missing fields — older iperf3 versions on PVE
// sometimes ship a slightly different schema.
func parseIperf3JSON(raw string, res *BandwidthTestResult) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("leere Antwort vom iperf3-Client")
	}

	type sumKind struct {
		BitsPerSecond float64 `json:"bits_per_second"`
		Bytes         int64   `json:"bytes"`
		Retransmits   int     `json:"retransmits"`
	}
	type endStruct struct {
		SumSent     sumKind `json:"sum_sent"`
		SumReceived sumKind `json:"sum_received"`
		Sum         sumKind `json:"sum"` // UDP path
	}

	var parsed struct {
		End   endStruct `json:"end"`
		Error string    `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return err
	}
	if parsed.Error != "" {
		return fmt.Errorf("iperf3-Fehler: %s", parsed.Error)
	}

	pick := parsed.End.SumSent
	if pick.BitsPerSecond == 0 {
		pick = parsed.End.SumReceived
	}
	if pick.BitsPerSecond == 0 {
		pick = parsed.End.Sum
	}
	res.BitsPerSecond = pick.BitsPerSecond
	res.BytesTotal = pick.Bytes
	res.Retransmits = pick.Retransmits
	return nil
}

func shellEscape(s string) string {
	// We only ever pass hostnames/IPs we resolved ourselves (from node config),
	// but we still defend against shell metacharacters by single-quoting.
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
