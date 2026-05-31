// Package xlsxcfg provides the core library for converting Excel (.xlsx) sheets
// into Protocol Buffer-defined config data (JSON and protobuf binary output).
//
// The conversion pipeline is:
//  1. Parse .proto files at runtime → build TypeProvider
//  2. Read .xlsx → iterate sheets column-wise → sheetParser → rowParser → tokenReader
//  3. Produce Maps → JSON → dynamic proto messages → .json and/or .bytes output
package xlsxcfg

import (
	"context"
	"fmt"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TypeProvider looks up proto message descriptors by their short name (e.g. "HeroSheet").
// The name is intentionally misspelled — do not rename it.
//
// It is the primary bridge between the proto schema and the xlsx parser: the row
// parser uses it to find message descriptors so it can determine field types,
// walk nested messages, and build dynamic proto messages for output.
type TypeProvider interface {
	// MessageByName returns the MessageDescriptor for the given short message name
	// (e.g. "HeroSheetRow"), or nil if no such message exists across all loaded proto files.
	MessageByName(string) protoreflect.MessageDescriptor
}

// LoadProtoFiles compiles the given .proto files at runtime using protocompile,
// so no protoc binary or code-generation step is needed for user schemas.
//
// importPaths sets the directories where proto imports are resolved from
// (analogous to protoc's -I flag). fileNames are the .proto files to compile.
//
// Returns a TypeProvider that indexes all top-level messages from the compiled
// files (and their transitive imports) by short name.
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

// protoTypeProvider implements TypeProvider by indexing all message descriptors
// from compiled proto files by their short (unqualified) name.
type protoTypeProvider struct {
	// fds holds all compiled file descriptors (including transitive imports),
	// keyed by file path (e.g. "hero.proto"). Used for deduplication during loading.
	fds map[string]linker.File
	// ss maps short message name → MessageDescriptor for O(1) lookup.
	ss map[string]protoreflect.MessageDescriptor
}

// newTypeProvider builds a protoTypeProvider from the given compiled files.
// It recursively registers each file and its imports, then indexes all top-level
// messages by short name.
func newTypeProvider(files ...linker.File) (TypeProvider, error) {
	d := &protoTypeProvider{
		fds: make(map[string]linker.File, len(files)),
		ss:  make(map[string]protoreflect.MessageDescriptor, len(files)),
	}
	for _, fd := range files {
		if err := addFile(fd, d.fds); err != nil {
			return nil, err
		}
		// Index all top-level messages from this file by their short name.
		msgs := fd.Messages()
		for i := 0; i < msgs.Len(); i++ {
			t := msgs.Get(i)
			d.ss[string(t.Name())] = t
		}
	}
	return d, nil
}

// addFile recursively registers a compiled proto file and all its transitive
// imports into the fds map. It detects duplicate files with the same path but
// different contents (which would indicate a conflicting import).
func addFile(fd linker.File, fds map[string]linker.File) error {
	name := fd.Path()
	if existing, ok := fds[name]; ok {
		// already added this file
		if existing != fd {
			// doh! duplicate files provided
			return fmt.Errorf("given files include multiple copies of %s", name)
		}
		return nil
	}
	fds[name] = fd
	// Walk transitive imports so they are all registered.
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

// MessageByName returns the MessageDescriptor for the given short message name,
// or nil if not found.
func (p *protoTypeProvider) MessageByName(name string) protoreflect.MessageDescriptor {
	return p.ss[name]
}

// GetFieldDescriptor walks a dot-separated field path through a message descriptor
// and returns the leaf FieldDescriptor. Returns nil if any segment in the path does
// not resolve to a field.
//
// Map values are traversed into their value message type if present.
func GetFieldDescriptor(md protoreflect.MessageDescriptor, path ...string) protoreflect.FieldDescriptor {
	var fieldDesc protoreflect.FieldDescriptor
	for _, fieldName := range path {
		fieldDesc = md.Fields().ByName(protoreflect.Name(fieldName))
		if fieldDesc == nil {
			return nil
		}
		// Descend into nested messages for the next path segment.
		if fieldDesc.IsMap() {
			md = fieldDesc.MapValue().Message()
		} else if m := fieldDesc.Message(); m != nil {
			md = m
		}
	}
	return fieldDesc
}

// IsStrField reports whether the leaf field at the given path is a string type.
// Delegates to GetFieldDescriptor for the path walk.
func IsStrField(md protoreflect.MessageDescriptor, path ...string) bool {
	fd := GetFieldDescriptor(md, path...)
	return fd != nil && fd.Kind() == protoreflect.StringKind
}
