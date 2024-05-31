package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/lemon-1997/dynamic-proxy"
	pb "github.com/lemon-1997/dynamic-proxy/examples/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	go grpcServer()
	httpServer()
}

func grpcServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	reflection.Register(s)
	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to grpc serve: %v", err)
	}
}

func httpServer() {
	proxy := dynamic_proxy.NewProxy()
	http.HandleFunc("/", proxy.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to http serve: %v", err)
	}
}
