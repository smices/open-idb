// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"time"
)

type backgroundLimiter struct {
	slots   chan struct{}
	timeout time.Duration
}

func newBackgroundLimiter(maxConcurrent int, timeout time.Duration) *backgroundLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &backgroundLimiter{
		slots:   make(chan struct{}, maxConcurrent),
		timeout: timeout,
	}
}

func (l *backgroundLimiter) do(ctx context.Context, operation func(context.Context) error) error {
	release, err := l.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	operationCtx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()
	return operation(operationCtx)
}

func (l *backgroundLimiter) acquire(ctx context.Context) (func(), error) {
	acquireCtx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()
	select {
	case l.slots <- struct{}{}:
		return func() { <-l.slots }, nil
	case <-acquireCtx.Done():
		return nil, acquireCtx.Err()
	}
}
