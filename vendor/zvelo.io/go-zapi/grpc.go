package zapi

import (
	"context"
	"crypto/tls"
	"io"

	"github.com/golang/protobuf/ptypes/empty"

	"golang.org/x/oauth2"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/grpclb" // register the grpclb balancer
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"

	"zvelo.io/go-zapi/internal/zvelo"
	msg "zvelo.io/msg/msgpb"
)

// A GRPCv1Client implements msg.APIv1Client as well as an io.Closer that, if
// closed, will close the underlying gRPC connection.
type GRPCv1Client interface {
	msg.APIv1Client
	io.Closer
}

type grpcV1Client struct {
	options *options
	client  msg.APIv1Client
	io.Closer
}

// A GRPCv1Dialer is used to simplify connecting to zveloAPI with the correct
// options. grpc DialOptions will override the defaults.
type GRPCv1Dialer interface {
	Dial(context.Context, ...grpc.DialOption) (GRPCv1Client, error)
}

type grpcV1Dialer struct {
	options *options
}

func (d grpcV1Dialer) Dial(ctx context.Context, opts ...grpc.DialOption) (GRPCv1Client, error) {
	dialOpts := append([]grpc.DialOption{}, opts...)

	if d.options.withoutTLS {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		// #nosec
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: d.options.tlsInsecureSkipVerify,
		})))
	}

	if d.options.TokenSource != nil {
		dialOpts = append(dialOpts,
			grpc.WithPerRPCCredentials(oauth.TokenSource{
				TokenSource: d.options,
			}),
		)
	}

	conn, err := grpc.DialContext(ctx, d.options.grpcTarget, dialOpts...)
	if err != nil {
		return nil, err
	}

	return grpcV1Client{
		Closer:  conn,
		client:  msg.NewAPIv1Client(conn),
		options: d.options,
	}, nil
}

// NewGRPCv1 returns a properly configured GRPCv1Dialer
func NewGRPCv1(ts oauth2.TokenSource, opts ...Option) GRPCv1Dialer {
	o := defaults(ts)
	for _, opt := range opts {
		opt(o)
	}

	return grpcV1Dialer{options: o}
}

func (c grpcV1Client) Query(ctx context.Context, in *msg.QueryRequests, opts ...grpc.CallOption) (*msg.QueryReplies, error) {
	zvelo.DebugContextOut(ctx, c.options.debug)
	header, trailer, opts := grpcMD(opts...)
	resp, err := c.client.Query(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header, *trailer)
	return resp, err
}

func (c grpcV1Client) Result(ctx context.Context, in *msg.RequestID, opts ...grpc.CallOption) (*msg.QueryResult, error) {
	zvelo.DebugContextOut(ctx, c.options.debug)
	header, trailer, opts := grpcMD(opts...)
	resp, err := c.client.Result(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header, *trailer)
	return resp, err
}

func (c grpcV1Client) Suggest(ctx context.Context, in *msg.Suggestion, opts ...grpc.CallOption) (*empty.Empty, error) {
	zvelo.DebugContextOut(ctx, c.options.debug)
	header, trailer, opts := grpcMD(opts...)
	resp, err := c.client.Suggest(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header, *trailer)
	return resp, err
}

func (c grpcV1Client) Stream(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (msg.APIv1_StreamClient, error) {
	if in == nil {
		in = &empty.Empty{}
	}

	zvelo.DebugContextOut(ctx, c.options.debug)
	header, trailer, opts := grpcMD(opts...)
	resp, err := c.client.Stream(ctx, in, opts...)
	zvelo.DebugMD(c.options.debug, *header, *trailer)
	return resp, err
}

func grpcMD(in ...grpc.CallOption) (header, trailer *metadata.MD, opts []grpc.CallOption) {
	for _, o := range in {
		if m, ok := o.(grpc.HeaderCallOption); ok {
			header = m.HeaderAddr
		}

		if m, ok := o.(grpc.TrailerCallOption); ok {
			trailer = m.TrailerAddr
		}
	}

	if header == nil {
		header = &metadata.MD{}
		opts = append(opts, grpc.Header(header))
	}

	if trailer == nil {
		trailer = &metadata.MD{}
		opts = append(opts, grpc.Trailer(trailer))
	}

	return
}
