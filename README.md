# Scrapollo

A CLI application that scrapes leads off of [apollo.io](https://apollo.io).

## Usage

```
scrapollo [flags]

Flags:
  -c, --cookie-file string       specify path to file containing cookies for your Apollo accounts
      --csv                      save output files in CSV format
  -d, --daily-limit int          daily limit for saving leads (default 500)
      --debug                    print debugging information
  -f, --fetch-credits            fetch credit usage for apollo accounts
  -H, --headless                 run browser in headless mode (default true)
  -h, --help                     help for scrapollo
  -i, --input string             path to file containing apollo accounts and scraping instructions
      --json                     save output files in JSON format
  -o, --output-dir string        specify path to output directory (default "./scrape-results")
      --stealth                  specify whether or not to inject stealth script at every page load
  -t, --tab string               specify the apollo.io tab from which leads will be scraped ('new', 'saved' or 'total') (default "new")
  -T, --timeout int              max time allowed for an operation (in seconds) (default 60)
  -v, --version                  version for scrapollo
      --vpn-args string          specify arguments to use with OpenVPN
      --vpn-configs-dir string   path to directory containing OpenVPN configuration files
      --vpn-credentials string   path to file containing OpenVPN credentials
```
