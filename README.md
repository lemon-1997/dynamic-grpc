# dynamic-grpc
Proxy HTTP requests to call gRPC services, use protoreflect to dynamically update the gRPC protocol without restart server.

## Feature
1. Support any http format conversion to protobuf(JSON,url query,url path,x-www-form-urlencoded).
2. Automatic upgrade when proto protocol is updated.
3. HTTP route is according to the
   [`google.api.http`](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto#L46)
   in your proto. 
4. No configuration required (use gRPC reflection).

## Examples

[https://github.com/lemon-1997/sqlboy/tree/main/examples](https://github.com/lemon-1997/dynamic-grpc/tree/main/examples)

## Blog post
If you want to learn more details about my motivation to write this and follow my steps in doing so, check out [my blog post](https://lemon-1997.pages.dev/post/project-grpc#more/) on the topic.

## Thanks
This repository is inspired by the following projects
- https://github.com/grpc-ecosystem/grpc-gateway
- https://github.com/jhump/protoreflect
