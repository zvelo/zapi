package poller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/urfave/cli"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/internal/zvelo"
	"zvelo.io/zapi/results"
)

type Handler interface {
	Result(context.Context, *results.Result) Requests
}

type HandlerFunc func(context.Context, *results.Result) Requests

func (f HandlerFunc) Result(ctx context.Context, results *results.Result) Requests {
	return f(ctx, results)
}

var _ Handler = HandlerFunc(nil)

// Requests is a map of request id to url
type Requests map[string]string

type Poller interface {
	Poll(ctx context.Context, requests Requests, fn Handler)
	Flags() []cli.Flag
	Once() bool
}

type poller struct {
	debug   *bool
	rest    *bool
	clients clients.Clients

	pollInterval time.Duration
	once         bool
}

func New(debug, rest *bool, clients clients.Clients) Poller {
	return &poller{
		debug:   debug,
		rest:    rest,
		clients: clients,
	}
}

func (p *poller) Flags() []cli.Flag {
	return []cli.Flag{
		cli.DurationFlag{
			Name:        "poll-interval",
			EnvVar:      "ZVELO_POLL_INTERVAL",
			Usage:       "repeatedly poll after this much time has elapsed until the request is marked as complete",
			Value:       1 * time.Second,
			Destination: &p.pollInterval,
		},
		cli.BoolFlag{
			Name:        "once",
			EnvVar:      "ZVELO_POLL_ONCE",
			Usage:       "make just a single poll request",
			Destination: &p.once,
		},
	}
}

func (p *poller) Once() bool {
	return p.once
}

func (p *poller) Poll(ctx context.Context, requests Requests, h Handler) {
	// do one poll immediately
	requests = p.pollRequests(ctx, requests, h)

	if p.once || len(requests) == 0 {
		return
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if requests = p.pollRequests(ctx, requests, h); len(requests) == 0 {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *poller) pollRequests(ctx context.Context, requests Requests, h Handler) Requests {
	stillPending := Requests{}

	for reqID, url := range requests {
		newRequests, err := p.pollRequest(ctx, reqID, url, h)
		if err != nil {
			zvelo.Errorf("%s\n", err)
			continue
		}

		for reqID, url := range newRequests {
			stillPending[reqID] = url
		}
	}

	return stillPending
}

func (p *poller) pollRequest(ctx context.Context, reqID, url string, h Handler) (Requests, error) {
	if *p.debug {
		if url == "" {
			fmt.Fprintf(os.Stderr, "polling for: %s\n", reqID) // #nosec
		} else {
			fmt.Fprintf(os.Stderr, "polling for: %s (%s)\n", url, reqID) // #nosec
		}
	}

	pollFn := p.pollGRPC
	if *p.rest {
		pollFn = p.pollREST
	}

	var result results.Result

	var err error
	result.QueryResult, result.PollTraceID, err = pollFn(ctx, reqID)

	if err != nil {
		return nil, err
	}

	newRequests := Requests{}

	if !zvelo.IsComplete(result.QueryResult) {
		newRequests[reqID] = url
	}

	for reqID, url := range h.Result(ctx, &result) {
		newRequests[reqID] = url
	}

	return newRequests, nil
}

func (p *poller) pollREST(ctx context.Context, reqID string) (result *msg.QueryResult, traceID string, err error) {
	return pollREST(ctx, p.clients.RESTv1(), reqID)
}

func pollREST(ctx context.Context, client zapi.RESTv1Client, reqID string) (result *msg.QueryResult, traceID string, err error) {
	var resp *http.Response
	result, err = client.Result(ctx, reqID, zapi.Response(&resp))
	if result != nil {
		traceID = resp.Header.Get(zapi.TraceHeader)
	}
	return result, traceID, err
}

func (p *poller) pollGRPC(ctx context.Context, reqID string) (*msg.QueryResult, string, error) {
	client, err := p.clients.GRPCv1(ctx)
	if err != nil {
		return nil, "<`1`>", err
	}

	req := msg.RequestID{RequestId: reqID}
	var header metadata.MD
	result, err := client.Result(ctx, &req, grpc.Header(&header))

	if *p.debug {
		zvelo.DebugHeader(header)
	}

	var traceID string
	if tids, ok := header[zapi.TraceHeader]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	return result, traceID, err
}
