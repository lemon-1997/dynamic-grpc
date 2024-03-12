package encoding

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/dynamic"
	"log/slog"
)

type jsonCodec struct {
	log          *slog.Logger
	marshalOpt   *jsonpb.Marshaler
	unmarshalOpt *jsonpb.Unmarshaler
}

func (jsonCodec) Marshal(msg *dynamic.Message) ([]byte, error) {
	return msg.MarshalJSONPB(&jsonpb.Marshaler{OrigName: true, EmitDefaults: true})
}

func (c jsonCodec) Unmarshal(data []byte, pathParams map[string]string, msg *dynamic.Message) error {
	for k, v := range pathParams {
		fd := msg.GetMessageDescriptor().FindFieldByName(k)
		if fd == nil {
			continue
		}
		val := decodeFields(fd, v)
		if val == nil {
			continue
		}
		if err := msg.TrySetFieldByName(k, val); err != nil {
			c.log.Warn("unmarshal set field fail", "field", k, "err", err)
		}
	}
	return msg.UnmarshalJSONPB(&jsonpb.Unmarshaler{AllowUnknownFields: true}, data)
}

func (jsonCodec) Subtype() string {
	return JsonSubType
}
