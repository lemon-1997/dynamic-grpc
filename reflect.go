package dynamic_proxy

import (
	"context"
	"fmt"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"log/slog"
	"net/http"
)

type ReflectClient struct {
	log    *slog.Logger
	conn   *grpc.ClientConn
	stub   grpcdynamic.Stub
	cancel context.CancelFunc
	router Router
}

func NewReflectClient(ctx context.Context, target string, log *slog.Logger, opts []grpc.DialOption) (*ReflectClient, error) {
	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %v", err)
	}
	stub := grpcdynamic.NewStub(conn)
	c, cancel := context.WithCancel(context.Background())
	g := &ReflectClient{
		log:    log,
		conn:   conn,
		stub:   stub,
		cancel: cancel,
	}
	g.watch(c)
	return g, nil
}

func (c *ReflectClient) MethodParams(method, path string) (*desc.MethodDescriptor, map[string]string) {
	params, extra, ok := c.router.Match(method, path)
	if !ok {
		return nil, nil
	}
	if md, ok := extra.(*desc.MethodDescriptor); ok {
		return md, params
	}
	return nil, nil
}

func (c *ReflectClient) Invoke(ctx context.Context, method *desc.MethodDescriptor, req *dynamic.Message) (*dynamic.Message, metadata.MD, error) {
	if method.IsServerStreaming() || method.IsClientStreaming() {
		return nil, nil, fmt.Errorf("failed to invoke stream")
	}
	md := metadata.New(make(map[string]string))
	res, err := c.stub.InvokeRpc(ctx, method, req, grpc.Header(&md))
	if err != nil {
		return nil, nil, err
	}
	dm := dynamic.NewMessage(method.GetOutputType())
	if err = dm.ConvertFrom(res); err != nil {
		return nil, nil, fmt.Errorf("conver output message error: %v", err)
	}
	return dm, md, nil
}

func (c *ReflectClient) Close() error {
	c.cancel()
	return c.conn.Close()
}

// https://github.com/googleapis/googleapis/blob/master/google/api/http.proto#L46
func (c *ReflectClient) route() (Router, error) {
	client := grpcreflect.NewClientAuto(context.Background(), c.conn)
	services, err := client.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to ListServices: %v", err)
	}
	router := NewRouter()
	for _, srv := range services {
		srvDesc, err := client.ResolveService(srv)
		if err != nil {
			return nil, fmt.Errorf("failed to ResolveService: %v", err)
		}
		methods := srvDesc.GetMethods()
		for _, method := range methods {
			opts := method.GetMethodOptions()
			ext := proto.GetExtension(opts, annotations.E_Http)
			httpOpt, ok := ext.(*annotations.HttpRule)
			if !ok {
				continue
			}
			switch (httpOpt.GetPattern()).(type) {
			case *annotations.HttpRule_Get:
				err = router.Add(http.MethodGet, httpOpt.GetGet(), method)
			case *annotations.HttpRule_Put:
				err = router.Add(http.MethodPut, httpOpt.GetPut(), method)
			case *annotations.HttpRule_Post:
				err = router.Add(http.MethodPost, httpOpt.GetPost(), method)
			case *annotations.HttpRule_Delete:
				err = router.Add(http.MethodDelete, httpOpt.GetDelete(), method)
			case *annotations.HttpRule_Patch:
				err = router.Add(http.MethodPatch, httpOpt.GetPatch(), method)
			}
			if err != nil {
				c.log.Error("build route fail", "err", err)
				continue
			}
		}
	}
	return router, nil
}

func (c *ReflectClient) watch(ctx context.Context) {
	router, err := c.route()
	if err != nil {
		c.log.Error("update method fail", "err", err)
	}
	c.router = router
	go func() {
		//defer func() {
		//	if rec := recover(); rec != nil {
		//		log.Printf("panic: %v", rec)
		//	}
		//}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			c.conn.WaitForStateChange(ctx, c.conn.GetState())
			if c.conn.GetState() != connectivity.Ready {
				continue
			}
			router, err = c.route()
			if err != nil {
				c.log.Error("update method fail", "err", err)
				continue
			}
			c.router = router
			c.log.Info("update method", "target", c.conn.Target())
		}
	}()
}
