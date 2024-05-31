package dynamic_proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/lemon-1997/dynamic-proxy/encoding"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Proxy struct {
	opts proxyOptions
	srv  sync.Map
}

type ProxyOption func(*proxyOptions)

type PathExtractFunc func(string) (grpcTarget string, httpRoute string)

type ErrorDecodeFunc func(w http.ResponseWriter, err error)

type proxyOptions struct {
	log                   *slog.Logger
	timeout               time.Duration
	marshaler             *jsonpb.Marshaler
	unmarshaler           *jsonpb.Unmarshaler
	incomingHeaderMatcher runtime.HeaderMatcherFunc
	outgoingHeaderMatcher runtime.HeaderMatcherFunc
	pathExtract           PathExtractFunc
	errDecoder            ErrorDecodeFunc
	grpcOpts              []grpc.DialOption
}

func WithLogger(logger *slog.Logger) ProxyOption {
	return func(o *proxyOptions) {
		o.log = logger
	}
}

func WithMarshaler(m *jsonpb.Marshaler) ProxyOption {
	return func(o *proxyOptions) {
		o.marshaler = m
	}
}

func WithUnmarshaler(m *jsonpb.Unmarshaler) ProxyOption {
	return func(o *proxyOptions) {
		o.unmarshaler = m
	}
}

func WithIncomingHeaderMatcher(f runtime.HeaderMatcherFunc) ProxyOption {
	return func(o *proxyOptions) {
		o.incomingHeaderMatcher = f
	}
}

func WithOutgoingHeaderMatcher(f runtime.HeaderMatcherFunc) ProxyOption {
	return func(o *proxyOptions) {
		o.outgoingHeaderMatcher = f
	}
}

func WithPathExtract(f PathExtractFunc) ProxyOption {
	return func(o *proxyOptions) {
		o.pathExtract = f
	}
}

func WithTimeout(d time.Duration) ProxyOption {
	return func(o *proxyOptions) {
		o.timeout = d
	}
}

func WithErrDecode(f ErrorDecodeFunc) ProxyOption {
	return func(o *proxyOptions) {
		o.errDecoder = f
	}
}

func WithGrpcOpts(opts ...grpc.DialOption) ProxyOption {
	return func(o *proxyOptions) {
		o.grpcOpts = opts
	}
}

func NewProxy(opts ...ProxyOption) *Proxy {
	options := proxyOptions{
		log:                   slog.New(slog.NewTextHandler(os.Stdout, nil)),
		timeout:               time.Second * 10,
		marshaler:             &jsonpb.Marshaler{OrigName: true, EmitDefaults: true},
		unmarshaler:           &jsonpb.Unmarshaler{AllowUnknownFields: true},
		incomingHeaderMatcher: runtime.DefaultHeaderMatcher,
		outgoingHeaderMatcher: DefaultHeaderMatcher,
		pathExtract:           DefaultPathExtract,
		errDecoder:            DefaultHTTPError,
		grpcOpts: []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	}
	for _, o := range opts {
		o(&options)
	}
	encoding.Register(options.marshaler, options.unmarshaler, options.log)
	return &Proxy{
		opts: options,
	}
}

func (p *Proxy) Client(ctx context.Context, target string) (*ReflectClient, error) {
	client, ok := p.srv.Load(target)
	if ok {
		return client.(*ReflectClient), nil
	}
	c, err := NewReflectClient(ctx, target, p.opts.log, p.opts.grpcOpts)
	if err != nil {
		return nil, err
	}
	p.srv.Store(target, c)
	return c, nil
}

func (p *Proxy) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), p.opts.timeout)
		defer cancel()

		target, path := p.opts.pathExtract(r.URL.Path)
		if target == "" || path == "" {
			p.opts.log.Warn("path not found", "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		client, err := p.Client(ctx, target)
		if err != nil {
			p.opts.log.Warn("target not found", "target", target)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		md, params := client.MethodParams(r.Method, path)
		if md == nil {
			p.opts.log.Warn("path not found", "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		msg := dynamic.NewMessage(md.GetInputType())
		if err = RequestEncode(r, msg, params); err != nil {
			p.opts.log.Error("request encode", "err", err)
			p.opts.errDecoder(w, err)
			return
		}

		ctx = metadata.NewOutgoingContext(ctx, p.metadataFromHeaders(r.Header))
		resp, header, err := client.Invoke(ctx, md, msg)
		if err != nil {
			p.opts.log.Error("client invoke", "err", err)
			p.opts.errDecoder(w, err)
			return
		}

		h := p.HeadersFromMetadata(header)
		for k, vs := range h {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}

		if err = ResponseDecode(r, w, resp); err != nil {
			p.opts.log.Error("response decode", "err", err)
			p.opts.errDecoder(w, err)
			return
		}

		return
	}
}

func (p *Proxy) metadataFromHeaders(raw map[string][]string) metadata.MD {
	md := make(map[string][]string)
	for k, v := range raw {
		key, ok := p.opts.incomingHeaderMatcher(k)
		if !ok {
			continue
		}
		newKey := strings.ToLower(key)
		// https://github.com/grpc/grpc-go/blob/master/internal/transport/http2_server.go#L417
		if newKey == "connection" {
			continue
		}
		md[newKey] = v
	}
	return md
}

func (p *Proxy) HeadersFromMetadata(md metadata.MD) map[string][]string {
	header := make(map[string][]string)
	for k, vs := range md {
		if h, ok := p.opts.outgoingHeaderMatcher(k); ok {
			header[h] = vs
		}
	}
	return md
}

func DefaultHeaderMatcher(key string) (string, bool) {
	return fmt.Sprintf("%s%s", runtime.MetadataHeaderPrefix, key), true
}

// DefaultPathExtract 格式：/target/route*
func DefaultPathExtract(path string) (string, string) {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return "", ""
	}
	target := fmt.Sprintf("%s:50051", parts[1])
	route := strings.TrimPrefix(path, fmt.Sprintf("/%s", parts[1]))
	return target, route
}
