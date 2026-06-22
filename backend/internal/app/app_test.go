// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/smices/open-idb/internal/config"
	"go.uber.org/zap"
)

func TestRunReturnsAfterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener := newBlockingListener()
	errCh := make(chan error, 1)
	go func() {
		app, err := New(ctx, config.Config{
			HTTPAddr:        "127.0.0.1:0",
			DefaultLocale:   "en-US",
			ShutdownTimeout: time.Second,
		}, zap.NewNop())
		if err != nil {
			errCh <- err
			return
		}
		errCh <- app.serve(ctx, listener)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation")
	}
}

type blockingListener struct {
	closed chan struct{}
}

func newBlockingListener() *blockingListener {
	return &blockingListener{closed: make(chan struct{})}
}

func (l *blockingListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *blockingListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}

func (l *blockingListener) Addr() net.Addr {
	return stubAddr("stub")
}

type stubAddr string

func (a stubAddr) Network() string {
	return string(a)
}

func (a stubAddr) String() string {
	return string(a)
}
