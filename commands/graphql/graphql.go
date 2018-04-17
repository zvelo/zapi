package graphql

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/results"
	"zvelo.io/zapi/tokensourcer"
)

type cmd struct {
	debug, trace bool
	timeout      time.Duration
	clients      clients.Clients
	query        string
}

func (c *cmd) Flags() []cli.Flag {
	return append(c.clients.Flags(),
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &c.debug,
		},
		cli.DurationFlag{
			Name:        "timeout",
			EnvVar:      "ZVELO_TIMEOUT",
			Usage:       "maximum amount of time to wait for results to complete",
			Value:       15 * time.Minute,
			Destination: &c.timeout,
		},
		cli.StringFlag{
			Name: "content",
			Usage: "the graphql query to request" +
				" if you start the content with the letter @, the rest should be a file name to read the data from, or - if you want zapi to read the data from stdin.",
			Destination: &c.query,
		},
		cli.BoolFlag{
			Name:        "trace",
			EnvVar:      "ZVELO_TRACE",
			Usage:       "request a trace to be generated for each request",
			Destination: &c.trace,
		},
	)
}

func Command(appName string) cli.Command {
	var c cmd
	tokenSourcer := tokensourcer.New(appName, &c.debug, strings.Fields(zapi.DefaultScopes)...)
	c.clients = clients.New(tokenSourcer, &c.debug)

	return cli.Command{
		Name:   "graphql",
		Usage:  "make graphql query",
		Before: c.setup,
		Action: c.action,
		Flags:  c.Flags(),
	}
}

func (c *cmd) setup(_ *cli.Context) error {
	if c.query == "" || c.query == "@" {
		return errors.New("content is required")
	}

	switch {
	case c.query == "", c.query == "@":
		return errors.New("content is required")
	case c.query == "@-":
		// '@-' means we need to read from stdin
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(os.Stdin); err != nil {
			return err
		}
		c.query = buf.String()
	case c.query[0] == '@':
		// '@' is a filename that should be read for the content
		data, err := ioutil.ReadFile(c.query[1:])
		if err != nil {
			return err
		}
		c.query = string(data)
	}

	return nil
}

func (c *cmd) action(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var opts []zapi.CallOption

	if c.trace {
		opts = append(opts, zapi.WithHeader("x-client-trace-id", results.TracingTag().String()))
	}

	var result string
	if err := c.clients.RESTv1().GraphQL(ctx, c.query, &result, opts...); err != nil {
		return err
	}

	fmt.Println(result)

	return nil
}
