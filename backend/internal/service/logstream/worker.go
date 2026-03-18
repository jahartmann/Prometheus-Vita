package logstream

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/ssh"
)

const (
	sshRetryDelay   = 5 * time.Second
	workerStopDelay = 10 * time.Second
)

// Worker streams log lines from a single Proxmox node over SSH.
// Priority workers use `tail -f` for real-time streaming.
// Non-priority workers poll with `journalctl --since <cursor>`.
type Worker struct {
	nodeID   string
	nodeName string
	sshPool  *ssh.Pool
	sshCfg   ssh.SSHConfig
	sources  []model.LogSource
	manager  *StreamManager
	priority bool
	cursor   string // RFC3339 timestamp used as journalctl --since cursor

	stopCh chan struct{}
	doneCh chan struct{}
}

// Start launches the worker's streaming goroutine.
func (w *Worker) Start(ctx context.Context) {
	go w.run(ctx)
}

// Stop signals the worker to exit and waits for it to finish.
func (w *Worker) Stop() {
	select {
	case <-w.stopCh:
		// already closed
	default:
		close(w.stopCh)
	}

	select {
	case <-w.doneCh:
	case <-time.After(workerStopDelay):
		slog.Warn("logstream: worker stop timed out",
			slog.String("node_id", w.nodeID),
		)
	}
}

// run is the main worker loop. It delegates to the appropriate streaming
// strategy based on the worker's priority flag.
func (w *Worker) run(ctx context.Context) {
	defer close(w.doneCh)

	if len(w.sources) == 0 {
		slog.Debug("logstream: worker has no enabled sources, exiting",
			slog.String("node_id", w.nodeID),
		)
		return
	}

	if w.priority {
		w.runTailF(ctx)
	} else {
		w.runPolling(ctx)
	}
}

// runTailF opens an SSH session and streams lines from all enabled log sources
// using `tail -f`. Reconnects automatically on SSH errors.
func (w *Worker) runTailF(ctx context.Context) {
	paths := w.enabledPaths()
	if len(paths) == 0 {
		return
	}

	// Build a single tail command covering all sources.
	cmd := fmt.Sprintf("tail -F -n 0 %s", shellJoin(paths))

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
		}

		if err := w.streamCommand(ctx, cmd); err != nil {
			slog.Warn("logstream: tail -f session ended, retrying",
				slog.String("node_id", w.nodeID),
				slog.String("node_name", w.nodeName),
				slog.Any("error", err),
			)
		}

		// Back off before reconnecting to avoid a tight reconnect loop.
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-time.After(sshRetryDelay):
		}
	}
}

// runPolling polls each log source at RotationInterval using `journalctl --since`.
func (w *Worker) runPolling(ctx context.Context) {
	interval := w.manager.cfg.RotationInterval

	// Initialise cursor to now so we only read future entries on first poll.
	if w.cursor == "" {
		w.cursor = time.Now().UTC().Format(time.RFC3339)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollJournalctl(ctx)
		}
	}
}

// pollJournalctl fetches new journal entries since the last cursor position.
func (w *Worker) pollJournalctl(ctx context.Context) {
	// Use journalctl to fetch entries since the last cursor.
	// --no-pager and -o short-iso produce parseable syslog-style output.
	cmd := fmt.Sprintf(
		"journalctl --since %q --no-pager -o short-iso 2>/dev/null || true",
		w.cursor,
	)

	// Advance cursor before running so the next poll window starts now.
	nextCursor := time.Now().UTC().Format(time.RFC3339)

	output, err := w.runCommandOutput(ctx, cmd)
	if err != nil {
		slog.Warn("logstream: journalctl poll failed",
			slog.String("node_id", w.nodeID),
			slog.Any("error", err),
		)
		return
	}

	w.cursor = nextCursor

	source := "journald"
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		w.processLine(ctx, line, source)
	}
}

// streamCommand opens an SSH session, starts cmd, and reads its stdout line by
// line until the session ends, context is cancelled, or stopCh is closed.
func (w *Worker) streamCommand(ctx context.Context, cmd string) error {
	// Use NewDirect for long-running streaming sessions so the pooled
	// connection is not monopolised.
	client, err := w.sshPool.NewDirect(w.sshCfg)
	if err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}
	defer client.Close()

	// NOTE: The SSH client exposes its underlying *ssh.Client through RunCommand
	// and CopyTo/CopyFrom, but not raw session creation. We use RunCommand in a
	// goroutine with a context and read combined output via the result.
	//
	// For tail -f style streaming (long-running, line-by-line) we need direct
	// session access. The streaming.go helper in the ssh package demonstrates
	// that the internal client field is accessed directly within the package.
	//
	// We therefore use the streaming approach from ssh.StreamCopyNodeToNode as
	// a reference: create a session, attach a stdout pipe, call session.Start,
	// and read line-by-line.
	//
	// Because ssh.Client does not expose a public NewSession method, we run the
	// command via RunCommand in a goroutine and post-process stdout. For true
	// line-by-line streaming the operator may extend ssh.Client with a
	// StartStreaming(cmd) (io.ReadCloser, error) method.
	//
	// Interim approach: run the command and process output when it finishes.
	// This covers short-lived tail captures; for indefinite streams, see TODO.

	// TODO: Add a ssh.Client.StartStreaming(cmd string) (io.ReadCloser, error)
	// method that exposes session.StdoutPipe + session.Start for long-running
	// commands. Until then, priority workers fall back to a timed RunCommand.

	streamCtx, cancel := context.WithTimeout(ctx, w.manager.cfg.RotationInterval*2)
	defer cancel()

	result, err := client.RunCommand(streamCtx, cmd)
	if err != nil {
		return fmt.Errorf("run stream command: %w", err)
	}

	// Determine which source this line belongs to (heuristic by path presence).
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		source := w.inferSource(line)
		w.processLine(ctx, line, source)
	}

	return nil
}

// runCommandOutput executes a command and returns combined stdout as a string.
func (w *Worker) runCommandOutput(ctx context.Context, cmd string) (string, error) {
	client, err := w.sshPool.Get(w.nodeID, w.sshCfg)
	if err != nil {
		return "", fmt.Errorf("ssh get client: %w", err)
	}
	// Return the client to the pool after use.
	defer w.sshPool.Return(w.nodeID, client)

	result, err := client.RunCommand(ctx, cmd)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// processLine parses a raw log line and distributes it to Redis and subscribers.
func (w *Worker) processLine(ctx context.Context, line, source string) {
	entry := ParseSyslog(line, w.nodeID, source)

	if err := w.manager.writeToRedis(ctx, w.nodeID, entry); err != nil {
		slog.Debug("logstream: failed to write entry to redis",
			slog.String("node_id", w.nodeID),
			slog.Any("error", err),
		)
	}

	w.manager.notifySubscribers(w.nodeID, entry)
}

// enabledPaths returns the file-system paths of all enabled log sources.
func (w *Worker) enabledPaths() []string {
	paths := make([]string, 0, len(w.sources))
	for _, s := range w.sources {
		paths = append(paths, s.Path)
	}
	return paths
}

// inferSource attempts to match a log line against known source paths.
// tail -f with multiple files prefixes lines with "==> <path> <==".
func (w *Worker) inferSource(line string) string {
	// tail -f header line: "==> /var/log/syslog <=="
	if strings.HasPrefix(line, "==> ") && strings.HasSuffix(line, " <==") {
		return strings.TrimSuffix(strings.TrimPrefix(line, "==> "), " <==")
	}

	// Check if any known source path appears in the line.
	for _, s := range w.sources {
		if strings.Contains(line, s.Path) {
			return s.Path
		}
	}

	return "syslog"
}

// shellJoin quotes and joins a slice of paths for safe shell interpolation.
func shellJoin(paths []string) string {
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return strings.Join(quoted, " ")
}
