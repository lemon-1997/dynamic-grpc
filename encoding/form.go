package encoding

import (
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/dynamic"
	"log/slog"
	"net/url"
)

type formCodec struct {
	log          *slog.Logger
	marshalOpt   *jsonpb.Marshaler
	unmarshalOpt *jsonpb.Unmarshaler
}

func (formCodec) Marshal(_ *dynamic.Message) ([]byte, error) {
	panic("not implemented")
}

func (c formCodec) Unmarshal(data []byte, pathParams map[string]string, msg *dynamic.Message) error {
	vs, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	for k, v := range pathParams {
		vs.Set(k, v)
	}
	for k, v := range vs {
		if len(v) == 0 {
			continue
		}
		fd := msg.GetMessageDescriptor().FindFieldByName(k)
		if fd == nil {
			if c.unmarshalOpt.AllowUnknownFields {
				continue
			}
			return fmt.Errorf("message type %s has no known field named %s", msg.GetMessageDescriptor().GetFullyQualifiedName(), k)
		}
		val := decodeFields(fd, v)
		if val == nil {
			continue
		}
		if err = msg.TrySetFieldByName(k, val); err != nil {
			c.log.Warn("unmarshal set field fail", "field", k, "err", err)
		}
	}
	return nil
}

func (formCodec) Subtype() string {
	return FormSubType
}
