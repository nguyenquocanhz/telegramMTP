package mtp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gotd/td/tgerr"
)

func testQueue(cfg QueueConfig) *Queue {
	return newQueue(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func await(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("job did not finish")
		return nil
	}
}

func TestQueueKeepsOrderAndPaces(t *testing.T) {
	const gap = 40 * time.Millisecond
	q := testQueue(QueueConfig{Interval: gap, ChatInterval: gap})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go q.run(ctx)

	var order []int
	var times []time.Time
	var last <-chan error
	for i := 0; i < 4; i++ {
		last = q.Push(int64(100+i), func(context.Context) error {
			order = append(order, i)
			times = append(times, time.Now())
			return nil
		})
	}
	if err := await(t, last); err != nil {
		t.Fatal(err)
	}

	for i, got := range order {
		if got != i {
			t.Fatalf("order = %v, want 0..3", order)
		}
	}
	for i := 1; i < len(times); i++ {
		// Allow a little timer slack below the configured gap.
		if d := times[i].Sub(times[i-1]); d < gap-10*time.Millisecond {
			t.Errorf("gap %d = %v, want >= %v", i, d, gap)
		}
	}
}

func TestQueuePerChatGap(t *testing.T) {
	const chatGap = 150 * time.Millisecond
	q := testQueue(QueueConfig{Interval: time.Millisecond, ChatInterval: chatGap})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go q.run(ctx)

	stamp := func(at *time.Time) func(context.Context) error {
		return func(context.Context) error { *at = time.Now(); return nil }
	}
	var a1, b, a2 time.Time
	q.Push(1, stamp(&a1))
	q.Push(2, stamp(&b))
	if err := await(t, q.Push(1, stamp(&a2))); err != nil {
		t.Fatal(err)
	}

	if d := b.Sub(a1); d > chatGap/2 {
		t.Errorf("different chat waited %v, want well under %v", d, chatGap)
	}
	if d := a2.Sub(a1); d < chatGap-10*time.Millisecond {
		t.Errorf("same chat gap = %v, want >= %v", d, chatGap)
	}
}

func TestQueueRetriesAfterFloodWait(t *testing.T) {
	q := testQueue(QueueConfig{Interval: time.Millisecond, ChatInterval: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go q.run(ctx)

	calls := 0
	start := time.Now()
	err := await(t, q.Push(1, func(context.Context) error {
		calls++
		if calls == 1 {
			return tgerr.New(420, "FLOOD_WAIT_1")
		}
		return nil
	}))
	if err != nil {
		t.Fatalf("err = %v, want nil after retry", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	if waited := time.Since(start); waited < time.Second {
		t.Errorf("retried after %v, want >= 1s flood wait", waited)
	}
}

func TestQueueReportsErrorsFullAndShutdown(t *testing.T) {
	q := testQueue(QueueConfig{Size: 1})

	boom := errors.New("boom")
	first := q.Push(1, func(context.Context) error { return boom })
	if err := <-q.Push(1, func(context.Context) error { return nil }); !errors.Is(err, ErrQueueFull) {
		t.Errorf("second push err = %v, want ErrQueueFull", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { q.run(ctx); close(stopped) }()
	if err := await(t, first); !errors.Is(err, boom) {
		t.Errorf("job err = %v, want boom", err)
	}

	// A job still waiting at shutdown is failed with the context error.
	cancel()
	<-stopped
	pending := q.Push(1, func(context.Context) error { return nil })
	q.drain(context.Canceled)
	if err := await(t, pending); !errors.Is(err, context.Canceled) {
		t.Errorf("pending job err = %v, want context.Canceled", err)
	}
}
