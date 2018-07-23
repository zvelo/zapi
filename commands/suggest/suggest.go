package suggest

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc/metadata"

	zapi "zvelo.io/go-zapi"
	msg "zvelo.io/msg/msgpb"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/results"
	"zvelo.io/zapi/tokensourcer"
)

type cmd struct {
	debug, trace, rest bool
	timeout            time.Duration
	clients            clients.Clients
	categories         cli.StringSlice
	malicious          cli.StringSlice
	notMalicious       bool
	suggestion         msg.Suggestion
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
		cli.DurationFlag{
			Name:        "timeout",
			EnvVar:      "ZVELO_TIMEOUT",
			Usage:       "maximum amount of time to wait for results to complete",
			Value:       15 * time.Minute,
			Destination: &c.timeout,
		},
		cli.StringFlag{
			Name:        "url",
			Usage:       "url to make suggestion for",
			Destination: &c.suggestion.Url,
		},
		cli.StringSliceFlag{
			Name:  "category",
			Usage: "categories to suggest, may be repeated",
			Value: &c.categories,
		},
		cli.StringSliceFlag{
			Name:  "malicious-category",
			Usage: "malicious category to suggest, may be repeated",
			Value: &c.malicious,
		},
		cli.BoolFlag{
			Name:        "not-malicious",
			Usage:       "suggest that the url should not be considered malicious",
			Destination: &c.notMalicious,
		},
	)
}

func Command(appName string) cli.Command {
	var c cmd
	tokenSourcer := tokensourcer.New(appName, &c.debug, "zvelo.suggest")
	c.clients = clients.New(tokenSourcer, &c.debug)

	return cli.Command{
		Name:   "suggest",
		Usage:  "suggest new datasets for a url",
		Before: c.setup,
		Action: c.action,
		Flags:  c.Flags(),
	}
}

func (c *cmd) setup(_ *cli.Context) error {
	if c.suggestion.Url == "" {
		return errors.New("url is required")
	}

	var cats []msg.Category
	for _, catName := range c.categories {
		cat := msg.ParseCategory(catName)
		if cat == msg.UNKNOWN_CATEGORY {
			return errors.Errorf("invalid category: %s", catName)
		}
		cats = append(cats, cat)
	}

	if len(cats) > 0 {
		if c.suggestion.Dataset == nil {
			c.suggestion.Dataset = &msg.Dataset{}
		}

		c.suggestion.Dataset.Categorization = &msg.Dataset_Categorization{
			Value: cats,
		}
	}

	if len(c.malicious) > 0 && c.notMalicious {
		return errors.New("can't suggest both malicious categories and that the url is not malicious")
	}

	var malcats []msg.Category
	for _, catName := range c.malicious {
		cat := msg.ParseCategory(catName)
		if cat == msg.UNKNOWN_CATEGORY {
			return errors.Errorf("invalid category: %s", catName)
		}
		malcats = append(malcats, cat)
	}

	if len(malcats) > 0 {
		if c.suggestion.Dataset == nil {
			c.suggestion.Dataset = &msg.Dataset{}
		}

		c.suggestion.Dataset.Malicious = &msg.Dataset_Malicious{
			Category: malcats,
		}
	}

	if c.notMalicious {
		if c.suggestion.Dataset == nil {
			c.suggestion.Dataset = &msg.Dataset{}
		}

		c.suggestion.Dataset.Malicious = &msg.Dataset_Malicious{
			Category: []msg.Category{},
		}
	}

	if c.suggestion.Dataset == nil {
		return errors.New("nothing to suggest")
	}

	return nil
}

func (c *cmd) action(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if c.rest {
		return c.suggestREST(ctx)
	}

	return c.suggestGRPC(ctx)
}

func (c *cmd) suggestGRPC(ctx context.Context) error {
	if c.trace {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-client-trace-id", results.TracingTag().String())
	}

	client, err := c.clients.GRPCv1(ctx)
	if err != nil {
		return err
	}

	if _, err = client.Suggest(ctx, &c.suggestion); err != nil {
		return err
	}

	return nil
}

func (c *cmd) suggestREST(ctx context.Context) error {
	var opts []zapi.CallOption

	if c.trace {
		opts = append(opts, zapi.WithHeader("x-client-trace-id", results.TracingTag().String()))
	}

	if err := c.clients.RESTv1().Suggest(ctx, &c.suggestion, opts...); err != nil {
		return err
	}

	return nil
}
