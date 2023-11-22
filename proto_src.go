package xlsxcfg

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/types/descriptorpb"
)

type TypeProvidor interface {
	MessageByName(string) *desc.MessageDescriptor
}

func LoadProtoFiles(importPaths []string, fileNames ...string) (tp TypeProvidor, err error) {
	fileNames, err = protoparse.ResolveFilenames(importPaths, fileNames...)
	if err != nil {
		return
	}
	pr := protoparse.Parser{
		ImportPaths:      importPaths,
		InferImportPaths: len(importPaths) == 0,
		// IncludeSourceCodeInfo: true,
	}
	fds, err := pr.ParseFiles(fileNames...)
	if err != nil {
		err = fmt.Errorf("could not parse given proto files: %v", err)
		return
	}
	tp, err = newTypeProvidor(fds...)
	return
}

type protoTypeProvidor struct {
	fds map[string]*desc.FileDescriptor
	ss  map[string]*desc.MessageDescriptor
}

func newTypeProvidor(fds ...*desc.FileDescriptor) (TypeProvidor, error) {
	d := &protoTypeProvidor{
		fds: make(map[string]*desc.FileDescriptor, len(fds)),
		ss:  make(map[string]*desc.MessageDescriptor, len(fds)),
	}
	for _, fd := range fds {
		if err := addFile(fd, d.fds); err != nil {
			return nil, err
		}
		for _, t := range fd.GetMessageTypes() {
			d.ss[t.GetName()] = t
		}
	}
	return d, nil
}

func addFile(fd *desc.FileDescriptor, fds map[string]*desc.FileDescriptor) error {
	name := fd.GetName()
	if existing, ok := fds[name]; ok {
		// already added this file
		if existing != fd {
			// doh! duplicate files provided
			return fmt.Errorf("given files include multiple copies of %s", name)
		}
		return nil
	}
	fds[name] = fd
	for _, dep := range fd.GetDependencies() {
		if err := addFile(dep, fds); err != nil {
			return err
		}
	}
	return nil
}

func (p *protoTypeProvidor) MessageByName(name string) *desc.MessageDescriptor {
	return p.ss[name]
}

func IsStrField(md *desc.MessageDescriptor, path ...string) bool {
	var fieldDesc *desc.FieldDescriptor
	for _, fieldName := range path {
		fieldDesc = md.FindFieldByName(fieldName)
		if fieldDesc.IsMap() {
			md = fieldDesc.GetMapValueType().GetMessageType()
		} else if m := fieldDesc.GetMessageType(); m != nil {
			md = m
		}
	}
	if fieldDesc == nil {
		return false
	}
	return *(fieldDesc.AsFieldDescriptorProto().Type) == descriptorpb.FieldDescriptorProto_TYPE_STRING
}
