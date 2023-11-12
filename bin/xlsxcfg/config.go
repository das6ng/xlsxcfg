package main

import (
	_ "embed"
	"log"
	"os"
)

//go:embed xlsxcfg.yaml
var configExampleContent []byte

const defaultConfigFileName = "xlsxcfg.yaml"

func exportExampleConfigFile() {
	if err := os.WriteFile(defaultConfigFileName, configExampleContent, 0644); err != nil {
		log.Println("write '", defaultConfigFileName, "' failed:", err)
	}
}
