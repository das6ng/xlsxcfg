package main

import (
	"context"
	"log"
	"os"

	"github.com/das6ng/xlsxcfg/app"
	"github.com/das6ng/xlsxcfg/flagutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:                "xlsxcfg [flags] [xlsx files...]",
	Short:              "xlsxcfg is a config parser",
	Long:               `A parser that converts xlsx sheets to config data in various formats.`,
	Run:                run,
	SilenceErrors:      true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
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
		return
	}
	if len(args) == 0 {
		log.Fatalln("no xlsx files specified")
	}

	cfg := loadConfig()

	if err := flagutil.ApplyOverrides(os.Args[1:], knownFlags(cmd), cfg); err != nil {
		log.Fatalln("apply flag overrides failed:", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalln("invalid config:", err)
	}

	if err := app.Run(context.Background(), cfg, args); err != nil {
		log.Fatalln(err)
	}
}

func knownFlags(cmd *cobra.Command) map[string]bool {
	flags := map[string]bool{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		flags["--"+f.Name] = true
		if f.Shorthand != "" {
			flags["-"+f.Shorthand] = true
		}
	})
	return flags
}
