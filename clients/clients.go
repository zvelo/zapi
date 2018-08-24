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

func New(tokenSourcer tokensourcer.TokenSourcer, debug, insecureSkipVerify *bool) Clients {
	return &data{
		debug:              debug,
		insecureSkipVerify: insecureSkipVerify,
		TokenSourcer:       tokenSourcer,
	}
}

type data struct {
	// cached data
	restV1 zapi.RESTv1Client
	grpcV1 zapi.GRPCv1Client

	// passed to constructor
	tokensourcer.TokenSourcer
	debug              *bool
	insecureSkipVerify *bool

	// from flags
	restBaseURL, grpcTarget string
	noTLS                   bool
}

func (d *data) Flags() []cli.Flag {
	return append(d.TokenSourcer.Flags(),
		cli.StringFlag{
			Name:        "rest-base-url",
			EnvVar:      "ZVELO_REST_BASE_URL",
			Usage:       "base URL of the API endpoint",
			Value:       zapi.DefaultRestBaseURL,
			Destination: &d.restBaseURL,
		},
		cli.StringFlag{
			Name:        "grpc-target",
			EnvVar:      "ZVELO_GRPC_TARGET",
			Usage:       "target for gRPC in the form of scheme://authority/endpoint_name",
			Value:       zapi.DefaultGrpcTarget,
			Destination: &d.grpcTarget,
		},
		cli.BoolFlag{
			Name:        "no-tls",
			Usage:       "disable tls",
			Destination: &d.noTLS,
		},
	)
}

func (d *data) zapiOpts() []zapi.Option {
	zapiOpts := []zapi.Option{
		zapi.WithRestBaseURL(d.restBaseURL),
		zapi.WithGrpcTarget(d.grpcTarget),
	}

	if *d.debug {
		zapiOpts = append(zapiOpts, zapi.WithDebug(os.Stderr))
	}

	if *d.insecureSkipVerify {
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
