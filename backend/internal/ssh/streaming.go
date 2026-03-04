package ssh

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

const streamBufferSize = 4 * 1024 * 1024 // 4MB

// StreamCopyNodeToNode pipes a file from source node to target node via SSH.
// It reads from source stdout and writes to target stdin using a fixed 4MB buffer.
// The onProgress callback is called with bytes transferred so far.
func StreamCopyNodeToNode(
	ctx context.Context,
	sourceClient *Client,
	targetClient *Client,
	sourcePath string,
	targetPath string,
	onProgress func(bytesSent int64),
) (int64, error) {
	// Create source session for reading
	srcSession, err := sourceClient.client.NewSession()
	if err != nil {
		return 0, fmt.Errorf("create source session: %w", err)
	}
	defer srcSession.Close()

	srcStdout, err := srcSession.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("source stdout pipe: %w", err)
	}

	// Create target session for writing
	tgtSession, err := targetClient.client.NewSession()
	if err != nil {
		return 0, fmt.Errorf("create target session: %w", err)
	}
	defer tgtSession.Close()

	tgtStdin, err := tgtSession.StdinPipe()
	if err != nil {
		return 0, fmt.Errorf("target stdin pipe: %w", err)
	}

	// Start source command: cat the vzdump file
	if err := srcSession.Start(fmt.Sprintf("cat %q", sourcePath)); err != nil {
		return 0, fmt.Errorf("start source cat: %w", err)
	}

	// Start target command: write to target path
	if err := tgtSession.Start(fmt.Sprintf("cat > %q", targetPath)); err != nil {
		return 0, fmt.Errorf("start target cat: %w", err)
	}

	// Copy with progress tracking
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
		tgtStdin.Close()
		done <- struct {
			n   int64
			err error
		}{n, err}
	}()

	select {
	case <-ctx.Done():
		_ = srcSession.Signal(ssh.SIGTERM)
		_ = tgtSession.Signal(ssh.SIGTERM)
		return 0, ctx.Err()
	case result := <-done:
		if result.err != nil {
			return result.n, fmt.Errorf("stream copy: %w", result.err)
		}
		// Wait for source to finish
		if err := srcSession.Wait(); err != nil {
			return result.n, fmt.Errorf("source session wait: %w", err)
		}
		// Wait for target to finish
		if err := tgtSession.Wait(); err != nil {
			return result.n, fmt.Errorf("target session wait: %w", err)
		}
		return result.n, nil
	}
}
