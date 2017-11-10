package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg"
)

var (
	pollOnce       bool
	pollRequestIDs []string
)

func init() {
	cmd := cli.Command{
		Name:      "poll",
		Usage:     "poll for results with a request-id",
		ArgsUsage: "request_id [request_id...]",
		Before:    setupPoll,
		Action:    poll,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "once",
				EnvVar:      "ZVELO_POLL_ONCE",
				Usage:       "make just a single poll request",
				Destination: &pollOnce,
			},
		},
	}
	cmd.BashComplete = bashCommandComplete(cmd)
	app.Commands = append(app.Commands, cmd)
}

func setupPoll(c *cli.Context) error {
	pollRequestIDs = c.Args()

	if len(pollRequestIDs) == 0 {
		return errors.New("at least one request_id is required")
	}

	go resultHandler()

	return nil
}

func poll(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultWg.Add(len(pollRequestIDs))

	// do one poll immediately
	if err := pollReqIDs(ctx, nil); err != nil {
		return err
	}

	wait := func() error {
		// wait for the wait group to complete or the context to timeout
		go func() {
			resultWg.Wait()
			cancel()
		}()

		<-ctx.Done()
		return ctx.Err()
	}

	if pollOnce || len(pollRequestIDs) == 0 {
		return wait()
	}

	// now start polling with a ticker
	go pollHandler(ctx, nil)

	return wait()
}

type resultCallback func(context.Context, *msg.QueryResult) []string

func pollHandler(ctx context.Context, fn resultCallback) {
	for range time.Tick(pollInterval) {
		if err := pollReqIDs(ctx, fn); err != nil {
			errorf("%s\n", err)
		}
	}
}

func pollReqIDs(ctx context.Context, fn resultCallback) error {
	var stillPending []string

	for _, reqID := range pollRequestIDs {
		complete, newReqIDs, err := pollReqID(ctx, reqID, fn)
		if err != nil {
			return err
		}

		stillPending = append(stillPending, newReqIDs...)

		if !complete {
			stillPending = append(stillPending, reqID)
		}
	}

	pollRequestIDs = stillPending

	return nil
}

func pollReqID(ctx context.Context, reqID string, fn resultCallback) (bool, []string, error) {
	url := getReqIDData(reqID, "").url

	if debug {
		if url == "" {
			fmt.Fprintf(os.Stderr, "polling for: %s\n", reqID)
		} else {
			fmt.Fprintf(os.Stderr, "polling for: %s (%s)\n", url, reqID)
		}
	}

	pollFn := pollGRPC
	if rest {
		pollFn = pollREST
	}

	var err error
	result := queryResult{pollStart: time.Now()}
	result.QueryResult, result.pollTraceID, err = pollFn(ctx, reqID)

	if err != nil {
		return false, nil, err
	}

	var newReqIDs []string
	if fn != nil {
		newReqIDs = fn(ctx, result.QueryResult)
	}

	resultCh <- result

	return isComplete(result.QueryResult), newReqIDs, nil
}

func pollREST(ctx context.Context, reqID string) (result *msg.QueryResult, traceID string, err error) {
	var resp *http.Response
	result, err = restV1Client.Result(ctx, reqID, zapi.Response(&resp))
	if result != nil {
		traceID = resp.Header.Get("uber-trace-id")
	}
	return result, traceID, err
}

func pollGRPC(ctx context.Context, reqID string) (*msg.QueryResult, string, error) {
	req := msg.RequestID{RequestId: reqID}
	var header metadata.MD
	result, err := grpcV1Client.Result(ctx, &req, grpc.Header(&header))
	var traceID string
	if tids, ok := header["uber-trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}
	return result, traceID, err
}
