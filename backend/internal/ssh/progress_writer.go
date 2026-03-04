package ssh

import (
	"io"
	"sync/atomic"
	"time"
)

// ProgressWriter wraps an io.Writer and tracks bytes written.
type ProgressWriter struct {
	Writer     io.Writer
	OnProgress func(bytesSent int64)
	written    atomic.Int64
	startTime  time.Time
	started    atomic.Bool
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	if !pw.started.Swap(true) {
		pw.startTime = time.Now()
	}

	n, err := pw.Writer.Write(p)
	if n > 0 {
		total := pw.written.Add(int64(n))
		if pw.OnProgress != nil {
			pw.OnProgress(total)
		}
	}
	return n, err
}

// BytesWritten returns the total number of bytes written so far.
func (pw *ProgressWriter) BytesWritten() int64 {
	return pw.written.Load()
}

// SpeedBps returns the average transfer speed in bytes per second.
func (pw *ProgressWriter) SpeedBps() int64 {
	if !pw.started.Load() {
		return 0
	}
	elapsed := time.Since(pw.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return int64(float64(pw.written.Load()) / elapsed)
}
