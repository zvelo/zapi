package poll

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	zapi "zvelo.io/go-zapi"
	msg "zvelo.io/msg/msgpb"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/internal/zvelo"
	"zvelo.io/zapi/poller"
	"zvelo.io/zapi/results"
	"zvelo.io/zapi/tokensourcer"
)

type cmd struct {
	debug, trace, rest, json bool
	clients                  clients.Clients
	poller                   poller.Poller
	timeout                  time.Duration
	requests                 poller.Requests
}

func (c *cmd) Flags() []cli.Flag {
	flags := append(c.clients.Flags(), c.poller.Flags()...)
	return append(flags,
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &c.debug,
		},
		cli.BoolFlag{
			Name:        "trace",
			EnvVar:      "ZVELO_TRACE",
			Usage:       "request a trace to be generated for each request",
			Destination: &c.trace,
		},
		cli.DurationFlag{
			Name:        "timeout",
			EnvVar:      "ZVELO_TIMEOUT",
			Usage:       "maximum amount of time to wait for results to complete",
			Value:       15 * time.Minute,
			Destination: &c.timeout,
		},
		cli.BoolFlag{
			Name:        "rest",
			EnvVar:      "ZVELO_REST",
			Usage:       "Use REST instead of gRPC for api requests",
			Destination: &c.rest,
		},
		cli.BoolFlag{
			Name:        "json",
			EnvVar:      "ZVELO_JSON",
			Usage:       "Print raw JSON response",
			Destination: &c.json,
		},
	)
}

func Command(appName string) cli.Command {
	var c cmd
	tokenSourcer := tokensourcer.New(appName, &c.debug, strings.Fields(zapi.DefaultScopes)...)
	c.clients = clients.New(tokenSourcer, &c.debug)
	c.poller = poller.New(&c.debug, &c.rest, &c.trace, c.clients)

	return cli.Command{
		Name:      "poll",
		Usage:     "poll for results with a request-id",
		ArgsUsage: "request_id [request_id...]",
		Before:    c.setup,
		Action:    c.action,
		Flags:     c.Flags(),
	}
}

func (c *cmd) setup(cli *cli.Context) error {
	c.requests = poller.Requests{}

	for _, requestID := range cli.Args() {
		c.requests[requestID] = ""
	}

	if len(c.requests) == 0 {
		return errors.New("at least one request_id is required")
	}

	return nil
}

func (c *cmd) action(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	c.poller.Poll(ctx, c.requests, c)

	return nil
}

func (c *cmd) Result(ctx context.Context, result *msg.QueryResult) poller.Requests {
	if complete := zvelo.IsComplete(result); complete || c.debug {
		results.Print(result, c.json)
	}

	return nil
}
