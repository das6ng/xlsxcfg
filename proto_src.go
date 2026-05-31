// Package xlsxcfg converts Excel (.xlsx) sheets into Protocol Buffer-defined config data.
//
// Pipeline: parse .proto → build TypeProvider → optionally load constants →
// read .xlsx row-wise (or column-wise for transposed sheets) → parse rows into
// OrderedMaps → convert to dynamicpb.Messages → write output (JSON, msgpack, protobuf binary).
package xlsxcfg

import (
	"context"
	"fmt"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TypeProvider looks up proto message descriptors by short name (e.g. "HeroSheet").
// The row parser uses it to determine field types, walk nested messages, and build
// dynamic proto messages.
type TypeProvider interface {
	// MessageByName returns the MessageDescriptor for the given short name, or nil.
	MessageByName(string) protoreflect.MessageDescriptor
}

// LoadProtoFiles compiles .proto files at runtime using protocompile (no protoc needed).
// Returns a TypeProvider indexing all top-level messages from the compiled files
// and their transitive imports by short name.
func LoadProtoFiles(ctx context.Context, importPaths []string, fileNames ...string) (tp TypeProvider, err error) {
	compiler := protocompile.Compiler{
		// WithStandardImports resolves well-known types like google.protobuf.Any.
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: importPaths,
		}),
	}
	files, err := compiler.Compile(ctx, fileNames...)
	if err != nil {
		err = fmt.Errorf("could not parse given proto files: %v", err)
		return
	}
	tp, err = newTypeProvider(files...)
	return
}

// protoTypeProvider indexes message descriptors from compiled proto files by short name.
type protoTypeProvider struct {
	fds map[string]linker.File                    // keyed by file path, for deduplication
	ss  map[string]protoreflect.MessageDescriptor // short name → descriptor
}

// newTypeProvider builds a protoTypeProvider from compiled files, recursively
// registering each file and its imports, then indexing top-level messages.
func newTypeProvider(files ...linker.File) (TypeProvider, error) {
	d := &protoTypeProvider{
		fds: make(map[string]linker.File, len(files)),
		ss:  make(map[string]protoreflect.MessageDescriptor, len(files)),
	}
	for _, fd := range files {
		if err := addFile(fd, d.fds); err != nil {
			return nil, err
		}
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			t := msgs.Get(i)
			d.ss[string(t.Name())] = t
		}
	}
	return d, nil
}

// addFile recursively registers a compiled proto file and its transitive imports.
// Returns an error if the same file path appears with different contents.
func addFile(fd linker.File, fds map[string]linker.File) error {
	name := fd.Path()
	if existing, ok := fds[name]; ok {
		if existing != fd {
			return fmt.Errorf("given files include multiple copies of %s", name)
		}
		return nil
	}
	fds[name] = fd
	imports := fd.Imports()
	for i := 0; i < imports.Len(); i++ {
		dep := imports.Get(i).FileDescriptor
		if lf, ok := dep.(linker.File); ok {
			if err := addFile(lf, fds); err != nil {
				return err
			}
		}
	}
	return nil
}

// MessageByName returns the MessageDescriptor for the given short name, or nil.
func (p *protoTypeProvider) MessageByName(name string) protoreflect.MessageDescriptor {
	return p.ss[name]
}

// GetFieldDescriptor walks a dot-separated field path through a message descriptor
// and returns the leaf FieldDescriptor, or nil if any segment doesn't resolve.
// Map values are traversed into their value message type.
func GetFieldDescriptor(md protoreflect.MessageDescriptor, path ...string) protoreflect.FieldDescriptor {
	var fieldDesc protoreflect.FieldDescriptor
	for _, fieldName := range path {
		fieldDesc = md.Fields().ByName(protoreflect.Name(fieldName))
		if fieldDesc == nil {
			return nil
		}
		if fieldDesc.IsMap() {
			md = fieldDesc.MapValue().Message()
		} else if m := fieldDesc.Message(); m != nil {
			md = m
		}
	}
	return fieldDesc
}

// IsStrField reports whether the leaf field at the given path is a string type.
func IsStrField(md protoreflect.MessageDescriptor, path ...string) bool {
	fd := GetFieldDescriptor(md, path...)
	return fd != nil && fd.Kind() == protoreflect.StringKind
}
