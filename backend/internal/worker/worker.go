// SPDX-License-Identifier: MIT

// Package worker provides background job processing for sync scheduling
// and asynchronous audit log writes.
package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Config controls Worker behaviour.
type Config struct {
	// PollInterval is how often the scheduler checks for pending work.
	// Defaults to 30 seconds when zero.
	PollInterval time.Duration

	// MaxConcurrentSyncs is the total number of sync jobs that may run
	// simultaneously across all entities.  Defaults to 3 when zero.
	MaxConcurrentSyncs int

	// MaxConcurrentPerEntity limits how many sync jobs a single entity
	// may execute at the same time.  Defaults to 1 when zero.
	MaxConcurrentPerEntity int

	// AuditBufferSize is the capacity of the async audit event channel.
	// Defaults to 1024 when zero.
	AuditBufferSize int
}

// applyDefaults fills in zero-valued fields with sensible defaults.
func (c *Config) applyDefaults() {
	if c.PollInterval <= 0 {
		c.PollInterval = 30 * time.Second
	}
	if c.MaxConcurrentSyncs <= 0 {
		c.MaxConcurrentSyncs = 3
	}
	if c.MaxConcurrentPerEntity <= 0 {
		c.MaxConcurrentPerEntity = 1
	}
	if c.AuditBufferSize <= 0 {
		c.AuditBufferSize = 1024
	}
}

// Worker orchestrates background goroutines for sync scheduling and
// asynchronous audit processing.  Create one with New, then call Start
// to launch the background goroutines and Stop to shut them down.
type Worker struct {
	cfg       Config
	logger    *zap.Logger
	scheduler *Scheduler
	audit     *AuditProcessor

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a Worker.  The scheduler and audit processor are wired
// internally; callers supply the SyncRunner and AuditWriter that connect
// to the database.
func New(cfg Config, logger *zap.Logger, runner *SyncRunner, auditSvc AuditWriter) *Worker {
	cfg.applyDefaults()

	audit := NewAuditProcessor(auditSvc, cfg.AuditBufferSize, logger)
	scheduler := NewScheduler(runner, audit, cfg.MaxConcurrentSyncs, cfg.MaxConcurrentPerEntity, logger)

	return &Worker{
		cfg:       cfg,
		logger:    logger,
		scheduler: scheduler,
		audit:     audit,
	}
}

// Scheduler returns the underlying Scheduler so callers can trigger jobs.
func (w *Worker) Scheduler() *Scheduler { return w.scheduler }

// AuditProcessor returns the underlying AuditProcessor so callers can
// enqueue fire-and-forget audit events.
func (w *Worker) AuditProcessor() *AuditProcessor { return w.audit }

// Start launches all background goroutines.  It is safe to call only
// once; subsequent calls are no-ops.
func (w *Worker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)

	w.logger.Info("starting worker",
		zap.Duration("poll_interval", w.cfg.PollInterval),
		zap.Int("max_concurrent_syncs", w.cfg.MaxConcurrentSyncs),
		zap.Int("max_per_entity", w.cfg.MaxConcurrentPerEntity),
		zap.Int("audit_buffer", w.cfg.AuditBufferSize),
	)

	// Audit drain goroutine.
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.audit.Run(ctx)
	}()

	// Scheduler dispatch goroutine.
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.scheduler.Run(ctx)
	}()
}

// Stop cancels the worker context and blocks until all goroutines exit.
// It drains any remaining audit events before returning.
func (w *Worker) Stop() {
	if w.cancel == nil {
		return
	}
	w.logger.Info("stopping worker")
	w.cancel()
	w.wg.Wait()

	// Final drain of audit buffer after all goroutines have exited.
	w.audit.Drain()

	w.logger.Info("worker stopped")
}
