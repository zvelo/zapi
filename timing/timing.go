package timing

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/oauth2"
	"zvelo.io/go-zapi/tokensource"
	"zvelo.io/zapi/internal/zvelo"
)

var printf = zvelo.PrintfFunc(color.FgBlue, os.Stderr)

func Context(ctx context.Context, debug bool) context.Context {
	if !debug {
		return ctx
	}

	var start, dnsStart, connectStart, tlsStart, reqStart time.Time

	trace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			start = time.Now()
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			printf("* DNS Lookup: %v\n", time.Since(dnsStart))
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			printf("* TCP Connection: %v\n", time.Since(connectStart))
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			printf("* TLS Handshake: %v\n", time.Since(tlsStart))
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			reqStart = time.Now()
		},
		GotFirstResponseByte: func() {
			printf("* Server Processing: %v\n", time.Since(reqStart))
			printf("* Total: %v\n", time.Since(start))
		},
	}

	return httptrace.WithClientTrace(ctx, trace)
}

func TokenSource(src oauth2.TokenSource, debug bool) oauth2.TokenSource {
	if !debug {
		return src
	}

	var mu sync.Mutex
	var start time.Time

	return tokensource.Tracer(tokensource.Trace{
		GetToken: func() {
			mu.Lock()
			start = time.Now()
		},
		GotToken: func(token *oauth2.Token, err error) {
			printf("* Token: %v\n", time.Since(start))
			mu.Unlock()
		},
	}, src)
}
