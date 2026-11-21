package mtp

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gotd/td/tgerr"
)

// Queue defaults. Telegram allows a bot roughly 30 messages per second
// overall and one per second into the same chat; the defaults stay under
// both.
const (
	DefaultQueueSize     = 1024
	DefaultQueueInterval = 50 * time.Millisecond
	DefaultChatInterval  = time.Second

	// queueMaxFloodWaits bounds how often one job is retried after
	// FLOOD_WAIT before its error is reported.
	queueMaxFloodWaits = 3
)

// ErrQueueFull is reported when the queue has no room for another job.
var ErrQueueFull = errors.New("mtp: message queue is full")

// QueueConfig tunes the outgoing queue. Zero values pick the defaults.
type QueueConfig struct {
	// Size is how many jobs may wait before Push reports ErrQueueFull.
	Size int
	// Interval is the minimum gap between any two sends.
	Interval time.Duration
	// ChatInterval is the minimum gap between two sends to the same chat.
	ChatInterval time.Duration
}

type queueJob struct {
	chatID int64
	run    func(ctx context.Context) error
	done   chan error
}

// Queue sends jobs one at a time, in order, no faster than Telegram's
// rate limits allow, and waits out FLOOD_WAIT errors instead of failing.
type Queue struct {
	jobs         chan queueJob
	interval     time.Duration
	chatInterval time.Duration
	log          *slog.Logger

	// Owned by the worker goroutine.
	lastSend time.Time
	lastChat map[int64]time.Time
}

func newQueue(cfg QueueConfig, log *slog.Logger) *Queue {
	if cfg.Size <= 0 {
		cfg.Size = DefaultQueueSize
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultQueueInterval
	}
	if cfg.ChatInterval <= 0 {
		cfg.ChatInterval = DefaultChatInterval
	}
	return &Queue{
		jobs:         make(chan queueJob, cfg.Size),
		interval:     cfg.Interval,
		chatInterval: cfg.ChatInterval,
		log:          log,
		lastChat:     make(map[int64]time.Time),
	}
}

// Push schedules fn. chatID is only used for per-chat pacing. The
// returned channel receives exactly one value: nil once fn succeeded,
// or the error. It is buffered, so ignoring it is fine.
func (q *Queue) Push(chatID int64, fn func(ctx context.Context) error) <-chan error {
	done := make(chan error, 1)
	select {
	case q.jobs <- queueJob{chatID: chatID, run: fn, done: done}:
	default:
		done <- ErrQueueFull
	}
	return done
}

// Len is the number of jobs waiting to be sent.
func (q *Queue) Len() int { return len(q.jobs) }

// run is the worker loop. It returns when ctx is cancelled, failing any
// jobs still waiting.
func (q *Queue) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			q.drain(ctx.Err())
			return
		case job := <-q.jobs:
			if err := q.pace(ctx, job.chatID); err != nil {
				job.done <- err
				continue
			}
			job.done <- q.exec(ctx, job)
			now := time.Now()
			q.lastSend = now
			q.lastChat[job.chatID] = now
		}
	}
}

func (q *Queue) drain(err error) {
	for {
		select {
		case job := <-q.jobs:
			job.done <- err
		default:
			return
		}
	}
}

// pace sleeps until both the global and the per-chat gap have passed.
func (q *Queue) pace(ctx context.Context, chatID int64) error {
	now := time.Now()
	ready := q.lastSend.Add(q.interval)
	if last, ok := q.lastChat[chatID]; ok {
		if t := last.Add(q.chatInterval); t.After(ready) {
			ready = t
		}
	}

	// Entries older than the per-chat gap no longer affect pacing.
	if len(q.lastChat) > 4096 {
		for id, t := range q.lastChat {
			if now.Sub(t) > q.chatInterval {
				delete(q.lastChat, id)
			}
		}
	}

	wait := ready.Sub(now)
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// exec runs the job, sleeping through FLOOD_WAIT and retrying.
func (q *Queue) exec(ctx context.Context, job queueJob) error {
	var err error
	for attempt := 0; attempt <= queueMaxFloodWaits; attempt++ {
		if err = job.run(ctx); err == nil {
			return nil
		}
		wait, isFlood := tgerr.AsFloodWait(err)
		if !isFlood || attempt == queueMaxFloodWaits {
			return err
		}
		q.log.Warn("flood wait, pausing queue", "wait", wait, "chat", job.chatID, "pending", q.Len())
		if waited, _ := tgerr.FloodWait(ctx, err); !waited {
			return ctx.Err()
		}
	}
	return err
}

// Queue returns the client's outgoing queue for custom jobs.
func (c *Client) Queue() *Queue { return c.queue }

// QueueMessage schedules a text message instead of sending it right
// away. Use it for bursts and broadcasts so the bot stays inside
// Telegram's rate limits. See Queue.Push for the returned channel.
func (c *Client) QueueMessage(chatID int64, text string, opts ...SendOption) <-chan error {
	return c.queue.Push(chatID, func(ctx context.Context) error {
		_, err := c.SendMessage(ctx, chatID, text, opts...)
		return err
	})
}

// QueuePhoto schedules an image the same way QueueMessage schedules text.
func (c *Client) QueuePhoto(chatID int64, p Photo, caption string, opts ...SendOption) <-chan error {
	return c.queue.Push(chatID, func(ctx context.Context) error {
		_, err := c.SendPhoto(ctx, chatID, p, caption, opts...)
		return err
	})
}
