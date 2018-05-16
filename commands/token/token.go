package token

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/coreos/go-oidc"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/zapi/tokensourcer"
)

type oidcToken struct {
	*oidc.IDToken
	Claims map[string]interface{}
}

var tokenTplStr = `
{{- if .AccessToken}}Access Token:  {{.AccessToken}}
{{end}}
{{- if .RefreshToken}}Refresh Token: {{.RefreshToken}}
{{end}}
{{- if .Expiry}}Expires At:    {{.Expiry}}
{{end -}}
{{- if idtoken .}}ID Token:      {{idtoken .}}
{{end}}
`

var idTokenTplStr = `Issuer:        {{.Issuer}}
Audience:      {{join .Audience}}
Subject:       {{.Subject}}
Issued At:     {{.IssuedAt}}
Claims:
  {{claims .Claims}}
`

var tokenTpl = template.Must(template.New("token").
	Funcs(template.FuncMap{
		"idtoken": func(i *oauth2.Token) string {
			if e := i.Extra("id_token"); e != nil {
				if s, ok := e.(string); ok {
					return s
				}
			}
			return ""
		},
	}).
	Parse(tokenTplStr))

var idTokenTpl = template.Must(template.New("id_token").
	Funcs(template.FuncMap{
		"join": func(i []string) string {
			return strings.Join(i, ", ")
		},
		"claims": func(i map[string]interface{}) string {
			var buf bytes.Buffer
			w := tabwriter.NewWriter(&buf, 0, 0, 0, ' ', 0)
			var keys []string
			for k := range i {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			for _, k := range keys {
				printClaim(w, "  ", k, i[k])
			}
			_ = w.Flush() // #nosec
			return strings.TrimSpace(buf.String())
		},
	}).
	Parse(idTokenTplStr))

func printClaim(w io.Writer, prefix, k string, v interface{}) {
	if v == nil || v == "" {
		return
	}

	m, ok := v.(map[string]interface{})
	if !ok {
		fmt.Fprintf(w, "%s%s: \t%v\n", prefix, k, v)
		return
	}

	fmt.Fprintf(w, "%s%s:\n", prefix, k)

	for mk, mv := range m {
		printClaim(w, prefix+"  ", mk, mv)
	}
}

type cmd struct {
	debug bool
	tokensourcer.TokenSourcer
	verifier *oidc.IDTokenVerifier
}

func (c *cmd) Flags() []cli.Flag {
	return append(c.TokenSourcer.Flags(),
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &c.debug,
		},
	)
}

func Command(appName string) cli.Command {
	var c cmd
	c.TokenSourcer = tokensourcer.New(appName, &c.debug, strings.Fields(zapi.DefaultScopes)...)

	return cli.Command{
		Name:   "token",
		Usage:  "retrieve a token for use elsewhere",
		Before: c.setup,
		Action: c.action,
		Flags:  c.Flags(),
	}
}

func (c *cmd) setup(_ *cli.Context) error {
	var err error
	c.verifier, err = c.Verifier(context.Background())
	return err
}

func (c *cmd) action(_ *cli.Context) error {
	tokensource := c.TokenSource()
	if tokensource == nil {
		return nil
	}

	token, err := tokensource.Token()
	if err != nil {
		return err
	}

	if err = tokenTpl.Execute(os.Stdout, token); err != nil {
		return err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if c.verifier == nil || !ok {
		return nil
	}

	var ot oidcToken

	if ot.IDToken, err = c.verifier.Verify(context.Background(), rawIDToken); err != nil {
		return err
	}

	if err = ot.IDToken.Claims(&ot.Claims); err != nil {
		return err
	}

	return idTokenTpl.Execute(os.Stdout, ot)
}
