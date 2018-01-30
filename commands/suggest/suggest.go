package suggest

import (
	"context"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/internal/zvelo"
	"zvelo.io/zapi/tokensourcer"
)

type cmd struct {
	debug, trace, rest bool
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
	tokenSourcer := tokensourcer.New(appName, &c.debug, &c.trace, "zvelo.suggest")
	c.clients = clients.New(tokenSourcer, &c.debug, &c.trace)

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
			c.suggestion.Dataset = &msg.DataSet{}
		}

		c.suggestion.Dataset.Categorization = &msg.DataSet_Categorization{
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
			c.suggestion.Dataset = &msg.DataSet{}
		}

		c.suggestion.Dataset.Malicious = &msg.DataSet_Malicious{
			Category: malcats,
		}
	}

	if c.notMalicious {
		if c.suggestion.Dataset == nil {
			c.suggestion.Dataset = &msg.DataSet{}
		}

		c.suggestion.Dataset.Malicious = &msg.DataSet_Malicious{
			Category: []msg.Category{},
		}
	}

	if c.suggestion.Dataset == nil {
		return errors.New("nothing to suggest")
	}

	return nil
}

func (c *cmd) action(_ *cli.Context) error {
	ctx := context.Background()

	if c.rest {
		return c.suggestREST(ctx)
	}

	return c.suggestGRPC(ctx)
}

func (c *cmd) suggestGRPC(ctx context.Context) error {
	client, err := c.clients.GRPCv1(ctx)
	if err != nil {
		return err
	}

	var header metadata.MD
	if _, err = client.Suggest(ctx, &c.suggestion, grpc.Header(&header)); err != nil {
		return err
	}

	if c.debug {
		zvelo.DebugHeader(header)
	}

	var traceID string
	if tids, ok := header[zapi.TraceHeader]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	complete(traceID)

	return nil
}

func (c *cmd) suggestREST(ctx context.Context) error {
	var resp *http.Response
	if err := c.clients.RESTv1().Suggest(ctx, &c.suggestion, zapi.Response(&resp)); err != nil {
		return err
	}

	complete(resp.Header.Get(zapi.TraceHeader))

	return nil
}

func complete(traceID string) {
	if traceID != "" {
		printf := zvelo.PrintfFunc(color.FgCyan, os.Stderr)
		printf("Trace ID: %s\n", traceID)
	}
}
