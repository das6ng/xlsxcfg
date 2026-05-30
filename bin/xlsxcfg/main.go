// Package main is the CLI entry point for xlsxcfg.
//
// xlsxcfg converts Excel (.xlsx) sheets into Protocol Buffer-defined config data
// (JSON and/or protobuf binary). It parses .proto files at runtime via protoreflect,
// so no protoc-gen step is needed for user proto schemas.
//
// Usage:
//
//	xlsxcfg [flags] <xlsx files...>
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
