// Copyright 2025 Abhisheke Acharya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/devsheke/scrapollo/internal/io"
	"github.com/devsheke/scrapollo/internal/logging"
	"github.com/devsheke/scrapollo/internal/models"
	"github.com/devsheke/scrapollo/internal/openvpn"
	"github.com/devsheke/scrapollo/internal/runner"
	"github.com/spf13/cobra"
)

const (
	APPNAME string = "scrapollo"
	VERSION string = "0.1.1"
)

var (
	dailyLimit, timeout                    int
	csvOut, jsonOut                        bool
	debug, fetchCredits, headless, stealth bool
	cookieFile, input, outputDir, tab      string
)

var vpnConfigs, vpnCredentialsFile, vpnArgs string

var rootCmd = &cobra.Command{
	Use:   APPNAME,
	Short: "Save and extract leads from apollo.io",
	Run: func(cmd *cobra.Command, args []string) {
		logging.Init(debug)

		var accounts []*models.Account
		if err := io.ReadRecords(input, &accounts); err != nil {
			exitOnError(err, 1)
		}

		runnerOpts := []runner.RunnerOpt{
			runner.Dailyimit(dailyLimit),
			runner.Debug(debug),
			runner.FetchCredits(fetchCredits),
			runner.Headless(headless),
			runner.OutputDir(outputDir),
			runner.Stealth(stealth),
			runner.Tab(tab),
			runner.Timeout(time.Duration(timeout) * time.Second),
		}

		if cookieFile != "" {
			runnerOpts = append(runnerOpts, runner.CookieFile(cookieFile))
		}

		if csvOut {
			runnerOpts = append(runnerOpts, runner.CsvOutput())
		} else if jsonOut {
			runnerOpts = append(runnerOpts, runner.JsonOutput())
		}

		if vpnConfigs != "" {
			vpn, err := openvpn.NewManager(vpnConfigs, vpnCredentialsFile, vpnArgs)
			if err != nil {
				exitOnError(err, 1)
			}

			runnerOpts = append(runnerOpts, runner.VpnManager(vpn))
		}

		r, err := runner.New(accounts, runnerOpts...)
		if err != nil {
			exitOnError(err, 1)
		}

		if err := r.Start(); err != nil {
			exitOnError(err, 1)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = VERSION

	rootCmd.Flags().
		StringVarP(&input, "input", "i", "", "path to file containing apollo accounts and scraping instructions")

	rootCmd.Flags().
		StringVarP(&outputDir, "output-dir", "o", "./scrape-results", "specify path to output directory")

	rootCmd.Flags().
		StringVarP(&cookieFile, "cookie-file", "c", "", "specify path to file containing cookies for your Apollo accounts")

	rootCmd.Flags().
		IntVarP(&dailyLimit, "daily-limit", "d", 500, "daily limit for saving leads")

	rootCmd.Flags().
		IntVarP(&timeout, "timeout", "T", 60, "max time allowed for an operation (in seconds)")

	rootCmd.Flags().BoolVar(&debug, "debug", false, "print debugging information")

	rootCmd.Flags().
		BoolVarP(&fetchCredits, "fetch-credits", "f", false, "fetch credit usage for apollo accounts")

	rootCmd.Flags().BoolVarP(&headless, "headless", "H", true, "run browser in headless mode")

	rootCmd.Flags().
		BoolVar(&stealth, "stealth", false, "specify whether or not to inject stealth script at every page load")

	rootCmd.Flags().BoolVar(&csvOut, "csv", false, "save output files in CSV format")

	rootCmd.Flags().BoolVar(&jsonOut, "json", false, "save output files in JSON format")

	rootCmd.Flags().
		StringVarP(&tab, "tab", "t", "new", "specify the apollo.io tab from which leads will be scraped ('new', 'saved' or 'total')")

	rootCmd.Flags().
		StringVar(&vpnConfigs, "vpn-configs-dir", "", "path to directory containing OpenVPN configuration files")

	rootCmd.Flags().
		StringVar(&vpnCredentialsFile, "vpn-credentials", "", "path to file containing OpenVPN credentials")

	rootCmd.Flags().
		StringVar(&vpnArgs, "vpn-args", "", "specify arguments to use with OpenVPN")

	if err := rootCmd.MarkFlagRequired("input"); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	rootCmd.MarkFlagsRequiredTogether("vpn-configs-dir", "vpn-credentials")

	rootCmd.MarkFlagsMutuallyExclusive("csv", "json")
	rootCmd.MarkFlagsOneRequired("csv", "json")
}

func exitOnError(err error, code int) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(code)
}
