package main

import (
	"context"
	"net"
	"testing"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg/mock"
)

// TestPollREST is a regression test for a previous version of
// pollREST which crashed due to an unchecked error return.
func TestPollREST(t *testing.T) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		l      net.Listener
		err    error
	)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	if l, err = net.Listen("tcp", "localhost:0"); err != nil {
		t.Error(err)
	}
	go func() {
		var err error
		if err = mock.APIv1().ServeTLS(ctx, l); err != nil {
			t.Error(err)
		}
	}()
	restV1Client = zapi.NewREST(nil, zapi.WithAddr(l.Addr().String()))
	// This should fail due to a TLS certificate error
	if _, _, err = pollREST(context.Background(), "22d29585-0204-406f-9941-ed15340c4c0f"); err == nil {
		t.Error("request which should have failed … succeeded instead")
	}
}
