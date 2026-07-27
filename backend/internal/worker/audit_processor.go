// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/smices/open-idb/internal/audit"
	"go.uber.org/zap"
)

// AuditWriter is the subset of audit.Service that the processor needs.
// Defining it as an interface allows testing without a real database.
type AuditWriter interface {
	Write(ctx context.Context, event audit.Event) error
}

// AuditProcessor provides a buffered, fire-and-forget channel for audit
// events.  A single background goroutine drains the queue and writes
// each event to the database through the synchronous AuditWriter.
//
// Degraded queue behavior: if the database write fails, the processor
// retries with exponential backoff.  If all retries are exhausted, the
// full event is logged as structured JSON at ERROR level so it can be
// recovered from log aggregation -- events never silently disappear.
type AuditProcessor struct {
	writer     AuditWriter
	events     chan audit.Event
	logger     *zap.Logger
	maxRetries int
	baseDelay  time.Duration
	wg         sync.WaitGroup
	limiter    *backgroundLimiter
}

// NewAuditProcessor creates an AuditProcessor with the given buffer
// capacity.  Call Run to start the drain goroutine.
func NewAuditProcessor(writer AuditWriter, bufferSize int, logger *zap.Logger) *AuditProcessor {
	return &AuditProcessor{
		writer:     writer,
		events:     make(chan audit.Event, bufferSize),
		logger:     logger,
		maxRetries: 3,
		baseDelay:  100 * time.Millisecond,
	}
}

// WriteAsync enqueues an audit event for asynchronous writing.  If the
// buffer is full, the caller blocks briefly (up to 1 second) before
// dropping the event with an error log.  This prevents silent loss
// under normal transient backpressure.
func (p *AuditProcessor) WriteAsync(event audit.Event) {
	select {
	case p.events <- event:
		return
	case <-time.After(1 * time.Second):
		// Buffer full for >1s -- log full event so it's not silently lost
		eventJSON, _ := json.Marshal(event)
		p.logger.Error("audit buffer full, event persisted to log",
			zap.String("action", event.Action),
			zap.String("entity_id", event.EntityID),
			zap.String("resource_type", event.ResourceType),
			zap.ByteString("event_json", eventJSON),
		)
	}
}

// Run starts the drain loop.  It blocks until ctx is cancelled, then
// returns.  Remaining events in the buffer are NOT drained here -- call
// Drain after the goroutine exits.
func (p *AuditProcessor) Run(ctx context.Context) {
	p.logger.Info("audit processor started")
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("audit processor stopping (context cancelled)")
			return
		case ev := <-p.events:
			p.writeWithRetry(ctx, ev)
		}
	}
}

// Drain writes all remaining events in the buffer synchronously.  This
// is called during graceful shutdown after the Run goroutine has exited,
// ensuring no events are lost.
func (p *AuditProcessor) Drain() {
	// Use a background context because the worker context is already
	// cancelled at this point.
	ctx := context.Background()
	drained := 0
	for {
		select {
		case ev := <-p.events:
			p.writeWithRetry(ctx, ev)
			drained++
		default:
			if drained > 0 {
				p.logger.Info("drained remaining audit events", zap.Int("count", drained))
			}
			return
		}
	}
}

// writeWithRetry attempts to write the event with exponential backoff.
// If all retries fail, the full event JSON is logged at ERROR level
// so it can be recovered from log aggregation systems.
func (p *AuditProcessor) writeWithRetry(ctx context.Context, ev audit.Event) {
	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms
			base := p.baseDelay * time.Duration(1<<(attempt-1))
			delay := base + time.Duration(rand.Int63n(max(1, int64(base/2))))
			select {
			case <-ctx.Done():
				p.persistToLog(ev, ctx.Err())
				return
			case <-time.After(delay):
			}
		}
		write := func(operationCtx context.Context) error {
			return p.writer.Write(operationCtx, ev)
		}
		var err error
		if p.limiter != nil {
			err = p.limiter.do(ctx, write)
		} else {
			err = write(ctx)
		}
		if err != nil {
			lastErr = err
			continue
		}
		return // success
	}
	// All retries exhausted -- persist to log for later recovery
	p.persistToLog(ev, lastErr)
}

// persistToLog writes the full event as structured JSON at ERROR level.
// This is the "degraded queue" fallback: events that cannot reach the
// database are preserved in log aggregation for manual or automated recovery.
func (p *AuditProcessor) persistToLog(ev audit.Event, cause error) {
	eventJSON, _ := json.Marshal(ev)
	p.logger.Error("audit event write failed permanently, persisted to log",
		zap.Error(cause),
		zap.String("action", ev.Action),
		zap.String("entity_id", ev.EntityID),
		zap.String("trace_id", ev.TraceID),
		zap.ByteString("event_json", eventJSON),
	)
}
