// Package convert provides functions to convert parsed xlsx data (map[string]any)
// into protobuf dynamic messages via protoreflect.
package convert

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// MapToProto populates a dynamic proto message from a map[string]any.
// It walks the map recursively, matching keys to proto field names (PascalCase),
// and sets values using protoreflect APIs — avoiding a JSON round-trip.
func MapToProto(data map[string]any, msg protoreflect.Message) error {
	md := msg.Descriptor()
	for k, v := range data {
		fd := md.Fields().ByName(protoreflect.Name(k))
		if fd == nil {
			continue
		}
		if err := setFieldValue(msg, fd, v); err != nil {
			return err
		}
	}
	return nil
}

// ScalarToProtoValue converts a Go value from the xlsx parser to a protoreflect.Value.
// The xlsx parser produces: int64 for all numeric fields, string for string fields.
func ScalarToProtoValue(fd protoreflect.FieldDescriptor, val any) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(val.(string))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.EnumKind:
		return protoreflect.ValueOfInt32(int32(val.(int64)))
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

func setRepeatedField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	list, ok := val.([]any)
	if !ok {
		return fmt.Errorf("expected list for repeated field %s, got %T", fd.Name(), val)
	}
	protoList := msg.Mutable(fd).List()
	for _, item := range list {
		if fd.Message() != nil {
			itemMsg := dynamicpb.NewMessage(fd.Message())
			itemMap, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("expected map for repeated message field %s item, got %T", fd.Name(), item)
			}
			if err := MapToProto(itemMap, itemMsg); err != nil {
				return err
			}
			protoList.Append(protoreflect.ValueOfMessage(itemMsg))
		} else {
			protoList.Append(ScalarToProtoValue(fd, item))
		}
	}
	return nil
}

func setMessageField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	subMap, ok := val.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map for message field %s, got %T", fd.Name(), val)
	}
	subMsg := dynamicpb.NewMessage(fd.Message())
	if err := MapToProto(subMap, subMsg); err != nil {
		return err
	}
	msg.Set(fd, protoreflect.ValueOfMessage(subMsg))
	return nil
}

func setScalarField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, val any) error {
	msg.Set(fd, ScalarToProtoValue(fd, val))
	return nil
}
