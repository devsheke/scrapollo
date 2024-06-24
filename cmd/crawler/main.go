package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const Version string = "0.0.1"

var (
	dailyLimit                    int
	debug, fetchCredits, headless bool
	input, outputDir              string
	vpnConfigs, vpnAuth, vpnArgs  string
)

var rootCmd = &cobra.Command{
	Use:          "apcr",
	Short:        "Save and extract leads from apollo.io",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogger(debug)

		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version

	rootCmd.Flags().
		StringVarP(&input, "input", "i", "", "path to input file containing apollo accounts")
	rootCmd.Flags().
		StringVarP(&outputDir, "output-dir", "o", "./scrape-results", "specify output directory")

	rootCmd.Flags().
		IntVar(&dailyLimit, "daily-limit", 500, "daily save limit (different from scrape limit)")

	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "print debug information")
	rootCmd.Flags().
		BoolVarP(&fetchCredits, "fetch-credits", "f", false, "fetch apollo credit usage before scraping")
	rootCmd.Flags().BoolVarP(&headless, "headless", "H", true, "run chrome in headless mode")

	rootCmd.Flags().
		StringVar(&vpnConfigs, "vpn-configs", "", "path to directory containing OpenVPN configs")
	rootCmd.Flags().
		StringVar(&vpnAuth, "vpn-auth", "", "path to OpenVPN credentials file")
	rootCmd.Flags().
		StringVar(&vpnArgs, "vpn-args", "", "specify additional OpenVPN args")

	_ = rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagsRequiredTogether("vpn-configs", "vpn-auth", "vpn-args")
}

func initLogger(debug bool) {
	var level zerolog.Level
	if debug {
		level = zerolog.DebugLevel
	} else {
		level = zerolog.InfoLevel
	}

	console := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "02/01/06 15:04-0700"}
	log.Logger = zerolog.New(console).With().Timestamp().Logger().Level(level)
}
