package clients

import (
	"context"
	"os"

	"github.com/urfave/cli"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/zapi/tokensourcer"
)

type Clients interface {
	Flags() []cli.Flag
	RESTv1() zapi.RESTv1Client
	GRPCv1(context.Context) (zapi.GRPCv1Client, error)
}

func New(tokenSourcer tokensourcer.TokenSourcer, debug *bool) Clients {
	return &data{
		debug:        debug,
		TokenSourcer: tokenSourcer,
	}
}

type data struct {
	// cached data
	restV1 zapi.RESTv1Client
	grpcV1 zapi.GRPCv1Client

	// passed to constructor
	tokensourcer.TokenSourcer
	debug *bool

	// from flags
	addr                  string
	tlsInsecureSkipVerify bool
	noTLS                 bool
}

func (d *data) Flags() []cli.Flag {
	return append(d.TokenSourcer.Flags(),
		cli.StringFlag{
			Name:        "addr",
			EnvVar:      "ZVELO_ADDR",
			Usage:       "address:port of the API endpoint",
			Value:       zapi.DefaultAddr,
			Destination: &d.addr,
		},
		cli.BoolFlag{
			Name:        "tls-insecure-skip-verify",
			Usage:       "disable certificate chain and host name verification of the connection to zveloAPI. this should only be used for testing, e.g. with mocks.",
			Destination: &d.tlsInsecureSkipVerify,
		},
		cli.BoolFlag{
			Name:        "no-tls",
			Usage:       "disable tls",
			Destination: &d.noTLS,
		},
	)
}

func (d *data) zapiOpts() []zapi.Option {
	var zapiOpts []zapi.Option

	zapiOpts = append(zapiOpts, zapi.WithAddr(d.addr))

	if *d.debug {
		zapiOpts = append(zapiOpts, zapi.WithDebug(os.Stderr))
	}

	if d.tlsInsecureSkipVerify {
		zapiOpts = append(zapiOpts, zapi.WithTLSInsecureSkipVerify())
	}

	if d.noTLS {
		zapiOpts = append(zapiOpts, zapi.WithoutTLS())
	}

	return zapiOpts
}

func (d *data) RESTv1() zapi.RESTv1Client {
	if d.restV1 == nil {
		d.restV1 = zapi.NewRESTv1(d.TokenSource(), d.zapiOpts()...)
	}

	return d.restV1
}

func (d *data) GRPCv1(ctx context.Context) (zapi.GRPCv1Client, error) {
	if d.grpcV1 != nil {
		return d.grpcV1, nil
	}

	grpcDialer := zapi.NewGRPCv1(d.TokenSource(), d.zapiOpts()...)

	var err error
	d.grpcV1, err = grpcDialer.Dial(ctx)
	return d.grpcV1, err
}
