package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/das6ng/xlsxcfg"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/runtime/protoiface"
)

var rootCmd = &cobra.Command{
	Use:   "xlsxcfg [flags] [xlsx files...]",
	Short: "xlsxcfg is a config parser",
	Long:  `A parser that converts xlsx sheets to Protocol Buffer defined config data.`,
	Run:   run,
}

var (
	configFile string
	configTmpl bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", defaultConfigFileName, "config file")
	rootCmd.PersistentFlags().BoolVar(&configTmpl, "example-config", false, "export an example config file here")
}

func run(cmd *cobra.Command, args []string) {
	if configTmpl {
		exportExampleConfigFile()
		os.Exit(0)
	}
	ctx := context.Background()
	cfg, err := xlsxcfg.ConfigFromFile(configFile)
	if err != nil {
		log.Fatalln("read config file failed:", err)
	}
	typeProvider, err := xlsxcfg.LoadProtoFiles(cfg.Proto.ImportPath, cfg.Proto.Files...)
	if err != nil {
		log.Fatalln("load proto files failed:", err)
	}
	sheetsData, err := xlsxcfg.LoadXlsxFiles(ctx, xlsxcfg.NewConfig(cfg, typeProvider), args...)
	if err != nil {
		log.Fatalln("parse xlsx files failed:", err)
	}

	msgFactory := dynamic.NewMessageFactoryWithDefaults()
	jsonMarshaler := &jsonpb.Marshaler{Indent: cfg.Output.JSONIndent}
	for sheet, data := range sheetsData {
		// marshal sheet data to json
		shtData := map[string]any{cfg.Sheet.ListFieldName: data}
		buf, err := json.Marshal(shtData)
		if err != nil {
			log.Fatalf("sheet[%s] marshal to json failed: %v\n", sheet, err)
		}
		// find sheet proto message and create an instance
		sheetTypeName := sheet + cfg.Sheet.TypeSuffix
		md := typeProvider.MessageByName(sheetTypeName)
		if md == nil {
			log.Printf("sheet[%s] cannot find proto: %s\n", sheet, sheetTypeName)
			continue
		}
		msg := msgFactory.NewMessage(md)
		// unmarshal sheet data to proto message
		if err := jsonpb.UnmarshalString(string(buf), msg); err != nil {
			log.Fatalf("sheet[%s] unmarshal json data failed: %v\n", sheet, err)
		}
		// write output file
		writeFile(cfg, sheet, msg, jsonMarshaler)
	}
}

func writeFile(cfg *xlsxcfg.ConfigFile, sheet string, msg protoiface.MessageV1, jm *jsonpb.Marshaler) {
	if cfg.Output.WriteBytes {
		// output proto bytes file
		buf, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("sheet[%s] marshal to bytes failed: %v\n", sheet, err)
		}
		outFile := path.Join(cfg.Output.Dir, sheet+".bytes")
		log.Println("writing file ...", outFile)
		if err = os.WriteFile(outFile, buf, 0644); err != nil {
			log.Printf("sheet[%s] write file[%s] failed: %s\n", sheet, outFile, err)
		}
	}
	if cfg.Output.WriteJSON {
		// output json file
		outJson, err := jm.MarshalToString(msg)
		if err != nil {
			log.Printf("sheet[%s] marshal pto json failed: %v\n", sheet, err)
		}
		outFile := path.Join(cfg.Output.Dir, sheet+".json")
		log.Println("writing file ...", outFile)
		if err = os.WriteFile(outFile, []byte(outJson), 0644); err != nil {
			log.Printf("sheet[%s] write file[%s] failed: %s\n", sheet, outFile, err)
		}
	}
}
