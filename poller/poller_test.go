package poller

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Error(err)
	}

	go func() {
		if serr := mock.APIv1().ServeTLS(ctx, l); serr != nil {
			t.Error(serr)
		}
	}()

	restV1Client := zapi.NewRESTv1(nil, zapi.WithRestBaseURL(l.Addr().String()))

	// This should fail due to a TLS certificate error
	if _, err = pollREST(context.Background(), restV1Client, "22d29585-0204-406f-9941-ed15340c4c0f", false, false); err == nil {
		t.Error("request which should have failed … succeeded instead")
	}
}
