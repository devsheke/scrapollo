package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/internal/io"
	"github.com/shadowbizz/apollo-crawler/internal/openvpn"
	"github.com/shadowbizz/apollo-crawler/internal/runner"
	"github.com/spf13/cobra"
)

const Version string = "0.0.1"

var (
	dailyLimit                                             int
	debug, fetchCredits, headless, saveProgress, json, csv bool
	input, outputDir, tab                                  string
	vpnConfigs, vpnAuth, vpnArgs                           string
)

var rootCmd = &cobra.Command{
	Use:   "scrapollo",
	Short: "Save and extract leads from apollo.io",
	Run: func(cmd *cobra.Command, args []string) {
		initLogger(debug)

		accounts, err := io.ReadAccountsFile(input)
		exitOnError(err)

		vpn, err := openvpn.NewVPN(vpnConfigs, vpnAuth, vpnArgs)
		exitOnError(err)

		opts := []runner.RunnerOpt{
			runner.Debug(debug),
			runner.FetchCredits(fetchCredits),
			runner.Headless(headless),
			runner.Dailyimit(dailyLimit),
			runner.OutputDir(outputDir),
			runner.SaveProgress(saveProgress),
			runner.SetTab(tab),
			runner.VPN(vpn),
		}

		if json {
			opts = append(opts, runner.JSONOutput())
		} else if csv {
			opts = append(opts, runner.CSVOutput())
		}

		r, err := runner.New(accounts, opts...)
		if err != nil {
			exitOnError(err)
		}

		exitOnError(r.Run())
	},
}

func main() {
	exitOnError(rootCmd.Execute())
}

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
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
		IntVarP(&dailyLimit, "daily-limit", "d", 500, "daily save limit (different from scrape limit)")

	rootCmd.Flags().BoolVar(&debug, "debug", false, "print debugging information")
	rootCmd.Flags().
		BoolVarP(&fetchCredits, "fetch-credits", "f", false, "fetch apollo credit usage before scraping")
	rootCmd.Flags().BoolVarP(&headless, "headless", "H", true, "run chrome in headless mode")
	rootCmd.Flags().
		BoolVarP(&saveProgress, "save-progress", "s", false, "save intermediary account information")

	rootCmd.Flags().BoolVar(&csv, "csv", false, "save output files in CSV format")
	rootCmd.Flags().BoolVar(&json, "json", false, "save output files in JSON format")

	rootCmd.Flags().
		StringVarP(&tab, "tab", "t", "", "sets the specified apollo tab before scraping ('new', 'saved' or 'total')")

	rootCmd.Flags().
		StringVar(&vpnConfigs, "vpn-configs", "", "path to directory containing OpenVPN configs")
	rootCmd.Flags().
		StringVar(&vpnAuth, "vpn-auth", "", "path to OpenVPN credentials file")
	rootCmd.Flags().
		StringVar(&vpnArgs, "vpn-args", "", "specify additional OpenVPN args")

	_ = rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagsRequiredTogether("vpn-configs", "vpn-auth")

	rootCmd.MarkFlagsMutuallyExclusive("csv", "json")
	rootCmd.MarkFlagsOneRequired("csv", "json")
}

func initLogger(debug bool) {
	var level zerolog.Level
	if debug {
		level = zerolog.DebugLevel
	} else {
		level = zerolog.InfoLevel
	}

	console := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "02/01/06 15:04:05-0700"}
	log.Logger = zerolog.New(console).With().Timestamp().Logger().Level(level)
}
