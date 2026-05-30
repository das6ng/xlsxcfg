package main

import (
	_ "embed"
	"log"
	"os"
	"strings"

	"github.com/das6ng/xlsxcfg"
)

//go:embed xlsxcfg.yaml
var configExampleContent []byte

const defaultConfigFileName = "xlsxcfg.yaml"

func exportExampleConfigFile() {
	if err := os.WriteFile(defaultConfigFileName, configExampleContent, 0644); err != nil {
		log.Println("write '", defaultConfigFileName, "' failed:", err)
	}
}

func loadConfig() *xlsxcfg.ConfigFile {
	_, err := os.Stat(configFile)
	if err == nil {
		cfg, err := xlsxcfg.ConfigFromFile(configFile)
		if err != nil {
			log.Fatalln("read config file failed:", err)
		}
		return cfg
	}
	if configFlagSet() {
		log.Fatalf("config file not found: %s\n", configFile)
	}
	return xlsxcfg.DefaultConfig()
}

func configFlagSet() bool {
	for _, arg := range os.Args[1:] {
		if arg == "-c" || arg == "--config" || strings.HasPrefix(arg, "-c=") || strings.HasPrefix(arg, "--config=") {
			return true
		}
	}
	return false
}
