package monitor

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestShutdownIsIdempotent(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := hub.Shutdown(ctx); err != nil {
		t.Fatalf("first Shutdown failed: %v", err)
	}
	if err := hub.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown should be a no-op, got %v", err)
	}
	if err := hub.Shutdown(ctx); err != nil {
		t.Fatalf("third Shutdown should be a no-op, got %v", err)
	}
}

func TestBroadcastAfterShutdownDoesNotBlock(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := hub.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		// Broadcasting after shutdown should return quickly even though
		// no Run loop is consuming the channel.
		hub.BroadcastMessage(WSMessage{Type: "test", Data: nil})
		close(done)
	}()
	select {
	case <-done:
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("BroadcastMessage blocked after Shutdown")
	}
}

func TestConcurrentRegisterUnregisterIsSafe(t *testing.T) {
	// Exercises the sync.Once guard on WSClient.send. Two paths can race to
	// close a client's send channel: the broadcast-drop fast path and the
	// regular unregister path. Without sync.Once the second close panics.
	hub := NewWSHub()
	go hub.Run()
	t.Cleanup(func() {
		_ = hub.Shutdown(context.Background())
	})

	const N = 100
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &WSClient{
				hub:  hub,
				send: make(chan []byte, 1),
			}
			hub.Register(client)
			// Race two paths that both try to close client.send.
			go hub.Unregister(client)
			client.closeSend()
		}()
	}
	wg.Wait()
}

func TestDroppedCounterIncreases(t *testing.T) {
	hub := NewWSHub()
	// Don't start Run() — broadcast channel will fill up and drop.
	for i := 0; i < 2000; i++ {
		hub.BroadcastMessage(WSMessage{Type: "metric", Data: i})
	}
	if hub.DroppedCount() == 0 {
		t.Fatal("expected at least one dropped message with a stalled hub, got 0")
	}
}
