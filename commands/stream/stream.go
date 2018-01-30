package stream

import (
	"context"
	"io"

	"github.com/urfave/cli"
	"zvelo.io/msg"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/results"
	"zvelo.io/zapi/tokensourcer"
)

type cmd struct {
	debug, trace, rest, json bool
	clients                  clients.Clients
}

func (c *cmd) Flags() []cli.Flag {
	return append(c.clients.Flags(),
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
	tokenSourcer := tokensourcer.New(appName, &c.debug, &c.trace, "zvelo.stream")
	c.clients = clients.New(tokenSourcer, &c.debug, &c.trace)

	return cli.Command{
		Name:   "stream",
		Usage:  "stream results from zveloAPI",
		Action: c.action,
		Flags:  c.Flags(),
	}
}

func (c *cmd) action(_ *cli.Context) error {
	ctx := context.Background()

	if c.rest {
		return c.streamREST(ctx)
	}

	return c.streamGRPC(ctx)
}

type streamClient interface {
	Recv() (*msg.QueryResult, error)
}

func (c *cmd) streamGRPC(ctx context.Context) error {
	client, err := c.clients.GRPCv1(ctx)
	if err != nil {
		return err
	}

	stream, err := client.Stream(ctx, nil)
	if err != nil {
		return err
	}

	return c.handle(stream)
}

func (c *cmd) streamREST(ctx context.Context) error {
	stream, err := c.clients.RESTv1().Stream(ctx)
	if err != nil {
		return err
	}

	return c.handle(stream)
}

func (c *cmd) handle(stream streamClient) error {
	for {
		result, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		results.Print(&results.Result{QueryResult: result}, c.json)
	}
}
