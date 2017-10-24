package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

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

	for _, reqID := range pollRequestIDs {
		if reqID != "" {
			reqIDs[reqID] = ""
		}
	}

	return nil
}

func poll(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return pollReqIDs(ctx)
}

func pollReqIDs(ctx context.Context) error {
	polling := map[string]string{}
	for reqID, url := range reqIDs {
		polling[reqID] = url
	}

	for len(polling) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}

		for reqID, url := range polling {
			complete, err := pollReqID(ctx, reqID, url)

			if err != nil {
				return err
			}

			if complete {
				delete(polling, reqID)
			}
		}

		if pollOnce {
			break
		}
	}

	return nil
}

func pollReqID(ctx context.Context, reqID, url string) (bool, error) {
	if debug {
		if url == "" {
			fmt.Fprintf(os.Stderr, "polling for: %s\n", reqID)
		} else {
			fmt.Fprintf(os.Stderr, "polling for: %s (%s)\n", url, reqID)
		}
	}

	var result *msg.QueryResult
	var traceID string
	var err error

	if rest {
		result, traceID, err = pollREST(ctx, reqID)
	} else {
		result, traceID, err = pollGRPC(ctx, reqID)
	}

	if err != nil {
		return false, err
	}

	fmt.Println()

	color.Set(color.FgCyan)

	if traceID != "" {
		fmt.Fprintf(os.Stderr, "Trace ID:           %s\n", traceID[:strings.Index(traceID, ":")])
	}

	if err := queryResultTpl.ExecuteTemplate(os.Stdout, "QueryResult", result); err != nil {
		color.Unset()
		return false, err
	}

	color.Unset()

	if result == nil || result.QueryStatus == nil {
		return false, nil
	}

	return result.QueryStatus.Complete, nil
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
