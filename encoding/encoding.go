package encoding

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/protobuf/types/descriptorpb"
	"log/slog"
	"strconv"
)

var (
	form  Codec
	json  Codec
	codec = map[string]Codec{}
)

const (
	JsonSubType = "json"
	FormSubType = "x-www-form-urlencoded"
)

type Codec interface {
	Marshal(v *dynamic.Message) ([]byte, error)
	Unmarshal(data []byte, params map[string]string, v *dynamic.Message) error
	Subtype() string
}

// TODO 支持xml html
func Register(marshalOpt *jsonpb.Marshaler, unmarshalOpt *jsonpb.Unmarshaler, log *slog.Logger) {
	form = &formCodec{
		log:          log,
		marshalOpt:   marshalOpt,
		unmarshalOpt: unmarshalOpt,
	}
	json = &jsonCodec{
		log:          log,
		marshalOpt:   marshalOpt,
		unmarshalOpt: unmarshalOpt,
	}
	codec[form.Subtype()] = form
	codec[json.Subtype()] = json
}

func CodecBySubtype(subtype string) Codec {
	return codec[subtype]
}

func decodeFields(fd *desc.FieldDescriptor, val string) interface{} {
	switch fd.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		vd := fd.GetEnumType().FindValueByName(val)
		if vd != nil {
			return vd.GetNumber()
		}
		return 0
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
		return false
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return []byte(val)
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return val[0]
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		if f, err := strconv.ParseFloat(val, 32); err == nil {
			return float32(f)
		}
		return float32(0)
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
		return float64(0)
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		if i, err := strconv.ParseInt(val, 10, 32); err == nil {
			return int32(i)
		}
		return int32(0)
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		if i, err := strconv.ParseUint(val, 10, 32); err == nil {
			return uint32(i)
		}
		return uint32(0)
	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
		return int64(0)
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		if i, err := strconv.ParseUint(val, 10, 64); err == nil {
			return i
		}
		return uint64(0)
	default:
		return nil
	}
}
