package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	zapi "zvelo.io/go-zapi"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var graphqlQuery string

func init() {
	cmd := cli.Command{
		Name:   "graphql",
		Usage:  "make graphql query",
		Before: graphQLSetup,
		Action: graphQL,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name: "content",
				Usage: "the graphql query to request" +
					" if you start the content with the letter @, the rest should be a file name to read the data from, or - if you want zapi to read the data from stdin.",
				Destination: &graphqlQuery,
			},
		},
	}
	cmd.BashComplete = bashCommandComplete(cmd)
	app.Commands = append(app.Commands, cmd)
}

func graphQLSetup(_ *cli.Context) error {
	if graphqlQuery == "" || graphqlQuery == "@" {
		return errors.New("content is required")
	}

	switch {
	case graphqlQuery == "", graphqlQuery == "@":
		return errors.New("content is required")
	case graphqlQuery == "@-":
		// '@-' means we need to read from stdin
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(os.Stdin); err != nil {
			return err
		}
		graphqlQuery = buf.String()
	case graphqlQuery[0] == '@':
		// '@' is a filename that should be read for the content
		data, err := ioutil.ReadFile(graphqlQuery[1:])
		if err != nil {
			return err
		}
		graphqlQuery = string(data)
	}

	return nil
}

func graphQL(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var resp *http.Response
	var result string
	if err := restV1Client.GraphQL(ctx, graphqlQuery, &result, zapi.Response(&resp)); err != nil {
		return err
	}

	traceID := resp.Header.Get("uber-trace-id")

	if traceID != "" {
		color.Set(color.FgCyan)
		fmt.Fprintf(os.Stderr, "Trace ID: %s\n", traceID[:strings.Index(traceID, ":")])
		color.Unset()
	}

	fmt.Println(result)

	return nil
}
