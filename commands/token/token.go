package token

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/coreos/go-oidc"
	"github.com/urfave/cli"

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
`

var idTokenTplStr = `Issuer:        {{.Issuer}}
Audience:      {{join .Audience}}
Subject:       {{.Subject}}
Issued At:     {{.IssuedAt}}
Claims:
  {{claims .Claims}}
`

var tokenTpl = template.Must(template.New("token").Parse(tokenTplStr))

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
				v := i[k]
				if v == nil || v == "" {
					continue
				}
				switch k {
				case "iat", "exp", "aud", "sub", "iss", "auth_time":
					continue
				}
				fmt.Fprintf(w, "  %s: \t%v\n", k, v)
			}
			_ = w.Flush()
			return strings.TrimSpace(buf.String())
		},
	}).
	Parse(idTokenTplStr))

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
	c.TokenSourcer = tokensourcer.New(appName, &c.debug)

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
