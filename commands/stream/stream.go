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

type streamClient interface {
	Recv() (*msg.QueryResult, error)
}

type constructor func() (streamClient, error)

func (c *cmd) action(_ *cli.Context) error {
	if c.rest {
		return c.handle(c.streamREST)
	}

	return c.handle(c.streamGRPC)
}

func (c *cmd) streamGRPC() (streamClient, error) {
	client, err := c.clients.GRPCv1(context.Background())
	if err != nil {
		return nil, err
	}

	return client.Stream(context.Background(), nil)
}

func (c *cmd) streamREST() (streamClient, error) {
	return c.clients.RESTv1().Stream(context.Background())
}

func (c *cmd) handle(client constructor) error {
	stream, err := client()
	if err != nil {
		return err
	}

	for {
		result, err := stream.Recv()

		if err == io.EOF {
			// try to reconnect

			if stream, err = client(); err != nil {
				return err
			}

			continue
		}

		if err != nil {
			return err
		}

		results.Print(&results.Result{QueryResult: result}, c.json)
	}
}
