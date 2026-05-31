// Package convert converts parsed xlsx data into protobuf dynamic messages.
package convert

import (
	"fmt"

	"github.com/das6ng/xlsxcfg"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// MapToProto populates a dynamic proto message from an OrderedMap or map[string]any,
// matching keys to proto field names directly — avoiding a JSON round-trip.
func MapToProto(data any, msg protoreflect.Message) error {
	md := msg.Descriptor()
	switch d := data.(type) {
	case *xlsxcfg.OrderedMap:
		for k, v := range d.All() {
			fd := md.Fields().ByName(protoreflect.Name(k))
			if fd == nil {
				continue
			}
			if err := setFieldValue(msg, fd, v); err != nil {
				return err
			}
		}
	case map[string]any:
		for k, v := range d {
			fd := md.Fields().ByName(protoreflect.Name(k))
			if fd == nil {
				continue
			}
			if err := setFieldValue(msg, fd, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// ScalarToProtoValue converts a Go value from the xlsx parser to a protoreflect.Value.
func ScalarToProtoValue(fd protoreflect.FieldDescriptor, val any) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(val.(string))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(val.(int64)))
	case protoreflect.EnumKind:
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(int32(val.(int64))))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(val.(int64))
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(val.(int64)))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(uint64(val.(int64)))
	case protoreflect.BoolKind:
		if n, ok := val.(int64); ok {
			return protoreflect.ValueOfBool(n != 0)
		}
		return protoreflect.ValueOfBool(val.(bool))
	case protoreflect.FloatKind:
		if n, ok := val.(int64); ok {
			return protoreflect.ValueOfFloat32(float32(n))
		}
		return protoreflect.ValueOfFloat32(val.(float32))
	case protoreflect.DoubleKind:
		if n, ok := val.(int64); ok {
			return protoreflect.ValueOfFloat64(float64(n))
		}
		return protoreflect.ValueOfFloat64(val.(float64))
	default:
		return protoreflect.ValueOfInt64(val.(int64))
	}
}

// setFieldValue dispatches value setting based on field type (list, message, or scalar).
func setFieldValue(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	if val == nil {
		return nil
	}
	switch {
	case fd.IsList():
		return setRepeatedField(msg, fd, val)
	case fd.Message() != nil:
		return setMessageField(msg, fd, val)
	default:
		return setScalarField(msg, fd, val)
	}
}

// setRepeatedField populates a repeated proto field from a []any slice.
func setRepeatedField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	list, ok := val.([]any)
	if !ok {
		return fmt.Errorf("expected list for repeated field %s, got %T", fd.Name(), val)
	}
	protoList := msg.Mutable(fd).List()
	for _, item := range list {
		if fd.Message() != nil {
			itemMsg := dynamicpb.NewMessage(fd.Message())
			if err := MapToProto(item, itemMsg); err != nil {
				return err
			}
			protoList.Append(protoreflect.ValueOfMessage(itemMsg))
		} else {
			protoList.Append(ScalarToProtoValue(fd, item))
		}
	}
	return nil
}

// setMessageField populates a nested message field via recursive MapToProto.
func setMessageField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	subMsg := dynamicpb.NewMessage(fd.Message())
	if err := MapToProto(val, subMsg); err != nil {
		return err
	}
	msg.Set(fd, protoreflect.ValueOfMessage(subMsg))
	return nil
}

// setScalarField sets a scalar proto field using ScalarToProtoValue.
func setScalarField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	msg.Set(fd, ScalarToProtoValue(fd, val))
	return nil
}
