package ssh

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	streamBufferSize     = 4 * 1024 * 1024 // 4MB
	streamShutdownWindow = 10 * time.Second
)

// StreamingSession is a long-running remote command whose stdout is read
// incrementally (line by line) rather than buffered to completion. Use it for
// never-terminating commands like `tail -F`. The caller MUST call Close() to
// terminate the remote command and release the SSH session.
type StreamingSession struct {
	session *ssh.Session
	// Stdout streams the command's standard output as it is produced.
	Stdout io.Reader
}

// Close signals the remote command to terminate and closes the session.
func (s *StreamingSession) Close() error {
	if s.session == nil {
		return nil
	}
	_ = s.session.Signal(ssh.SIGTERM)
	return s.session.Close()
}

// StartStreaming starts cmd on the remote host and returns a StreamingSession
// whose Stdout can be read incrementally. Unlike RunCommand, it does not buffer
// output or wait for the command to finish, so it works for indefinite streams
// (e.g. `tail -F`). The caller owns the returned session and must Close() it.
func (c *Client) StartStreaming(cmd string) (*StreamingSession, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create streaming session: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("streaming stdout pipe: %w", err)
	}
	if err := session.Start(cmd); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("start streaming command: %w", err)
	}
	return &StreamingSession{session: session, Stdout: stdout}, nil
}

// StreamCopyNodeToNode pipes a file from source node to target node via SSH.
// It reads from source stdout and writes to target stdin using a fixed 4MB buffer.
// The onProgress callback is called with bytes transferred so far.
//
// Cancellation safety: on ctx.Done() we signal both sessions, then wait a
// bounded amount of time for the background copy goroutine to exit before
// returning. Without this the goroutine could keep running, holding the
// SSH sessions open beyond the lifetime of the caller's context — and the
// previous version of this function could double-close tgtStdin (once from
// the copy goroutine, once via deferred session.Close) which would
// occasionally panic.
func StreamCopyNodeToNode(
	ctx context.Context,
	sourceClient *Client,
	targetClient *Client,
	sourcePath string,
	targetPath string,
	onProgress func(bytesSent int64),
) (int64, error) {
	srcSession, err := sourceClient.client.NewSession()
	if err != nil {
		return 0, fmt.Errorf("create source session: %w", err)
	}
	defer srcSession.Close()

	srcStdout, err := srcSession.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("source stdout pipe: %w", err)
	}

	tgtSession, err := targetClient.client.NewSession()
	if err != nil {
		return 0, fmt.Errorf("create target session: %w", err)
	}
	defer tgtSession.Close()

	tgtStdin, err := tgtSession.StdinPipe()
	if err != nil {
		return 0, fmt.Errorf("target stdin pipe: %w", err)
	}

	// Close tgtStdin at most once. CopyBuffer's goroutine wants to close it
	// on success; the cancellation path also wants to close it to unblock
	// any pending write. sync.Once guarantees both can be safely called.
	var stdinCloseOnce sync.Once
	closeStdin := func() {
		stdinCloseOnce.Do(func() {
			_ = tgtStdin.Close()
		})
	}

	if err := srcSession.Start(fmt.Sprintf("cat %s", ShellQuote(sourcePath))); err != nil {
		return 0, fmt.Errorf("start source cat: %w", err)
	}
	if err := tgtSession.Start(fmt.Sprintf("cat > %s", ShellQuote(targetPath))); err != nil {
		_ = srcSession.Signal(ssh.SIGTERM)
		return 0, fmt.Errorf("start target cat: %w", err)
	}

	var writer io.Writer = tgtStdin
	if onProgress != nil {
		writer = &ProgressWriter{
			Writer:     tgtStdin,
			OnProgress: onProgress,
		}
	}

	buf := make([]byte, streamBufferSize)
	done := make(chan struct {
		n   int64
		err error
	}, 1)

	go func() {
		n, err := io.CopyBuffer(writer, srcStdout, buf)
		closeStdin()
		done <- struct {
			n   int64
			err error
		}{n, err}
	}()

	select {
	case <-ctx.Done():
		// Tell the remote ends to stop and unblock our own goroutine by
		// closing stdin (it may be wedged on a write that no one is reading).
		_ = srcSession.Signal(ssh.SIGTERM)
		_ = tgtSession.Signal(ssh.SIGTERM)
		closeStdin()
		// Bounded wait for the copy goroutine so it does not leak.
		select {
		case <-done:
		case <-time.After(streamShutdownWindow):
			slog.Warn("ssh stream copy: goroutine did not exit within shutdown window — leaking",
				slog.String("source_path", sourcePath),
			)
		}
		return 0, ctx.Err()
	case result := <-done:
		if result.err != nil {
			return result.n, fmt.Errorf("stream copy: %w", result.err)
		}
		if err := srcSession.Wait(); err != nil {
			return result.n, fmt.Errorf("source session wait: %w", err)
		}
		if err := tgtSession.Wait(); err != nil {
			return result.n, fmt.Errorf("target session wait: %w", err)
		}
		return result.n, nil
	}
}
