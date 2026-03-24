package util

import (
	"log/slog"
	"runtime/debug"
)

// SafeGo launches a goroutine with panic recovery.
func SafeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panicked",
					slog.String("goroutine", name),
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		fn()
	}()
}
