package dynamic_proxy

import (
	"encoding/json"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/lemon-1997/dynamic-proxy/encoding"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"strings"
)

type Response struct {
	Status int32           `json:"status"`
	Msg    string          `json:"msg,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

func CodecForRequest(r *http.Request, name string) encoding.Codec {
	for _, accept := range r.Header[name] {
		codec := encoding.CodecBySubtype(contentSubtype(accept))
		if codec != nil {
			return codec
		}
	}
	return encoding.CodecBySubtype(encoding.JsonSubType)
}

func RequestEncode(req *http.Request, msg *dynamic.Message, pathParams map[string]string) error {
	switch req.Method {
	case http.MethodGet, http.MethodDelete:
		return QueryEncode(req, msg, pathParams)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return BodyEncode(req, msg, pathParams)
	}
	return nil
}

func QueryEncode(req *http.Request, msg *dynamic.Message, pathParams map[string]string) error {
	codec := encoding.CodecBySubtype(encoding.FormSubType)
	if err := codec.Unmarshal([]byte(req.URL.RawQuery), pathParams, msg); err != nil {
		return fmt.Errorf("codec unmarshal error: %v", err)
	}
	return nil
}

func BodyEncode(req *http.Request, msg *dynamic.Message, pathParams map[string]string) error {
	codec := CodecForRequest(req, "Content-Type")
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("read body error: %v", err)
	}
	defer req.Body.Close()
	if err = codec.Unmarshal(data, pathParams, msg); err != nil {
		return fmt.Errorf("codec unmarshal error: %v", err)
	}
	return nil
}

func ResponseDecode(r *http.Request, w http.ResponseWriter, msg *dynamic.Message) error {
	codec := CodecForRequest(r, "Accept")
	buf, err := codec.Marshal(msg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("failed to marshal output JSON: %v", err)
	}
	b, err := json.Marshal(Response{
		Status: 0,
		Data:   buf,
		Msg:    "ok",
	})
	if err != nil {
		return fmt.Errorf("failed to write response: %v", err)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/"+codec.Subtype())
	if _, err = w.Write(b); err != nil {
		return fmt.Errorf("failed to write response: %v", err)
	}
	return nil
}

func DefaultHTTPError(w http.ResponseWriter, err error) {
	grpcStatus := status.Convert(err)
	w.WriteHeader(runtime.HTTPStatusFromCode(grpcStatus.Code()))
	w.Write([]byte(grpcStatus.Message()))
}

func contentSubtype(contentType string) string {
	left := strings.Index(contentType, "/")
	if left == -1 {
		return ""
	}
	right := strings.Index(contentType, ";")
	if right == -1 {
		right = len(contentType)
	}
	if right < left {
		return ""
	}
	return contentType[left+1 : right]
}
