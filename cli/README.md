# webscanner (command line)

Lists what is publicly discoverable about a domain, its subdomains and the
pages on them, and where each one was found, from your own computer. It runs
the same discovery engines as the Web Scanner website, without the website's
limits on how much one scan may do.

Only scan domains you own or have permission to inspect.

## Install

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/anandhuremanan/Web-Scanner/main/cli/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/anandhuremanan/Web-Scanner/main/cli/install.ps1 | iex
```

Homebrew (macOS and Linux):

```sh
brew install anandhuremanan/tap/webscanner
```

Scoop (Windows):

```powershell
scoop install https://github.com/anandhuremanan/Web-Scanner/releases/latest/download/webscanner.json
```

The one-line installers above both download the latest release for your system from GitHub, check it
against the published checksums, and install it (Linux and macOS: in
`/usr/local/bin` if writable, otherwise `~/.local/bin`; Windows: in
`%LOCALAPPDATA%\Programs\webscanner`, added to your PATH). Set
`WEBSCANNER_VERSION` for a specific version or `INSTALL_DIR` for another
folder. You can also download a file from the
[releases page](https://github.com/anandhuremanan/Web-Scanner/releases)
yourself.

The programs are not signed, so Windows or macOS may warn the first time
you run one that you downloaded with a browser.

## Use

```sh
webscanner example.com                     # standard scan, results saved to a folder
webscanner -mode passive example.com       # never contacts the site
webscanner -mode full -pages 1000 example.com
webscanner -print urls -no-files example.com > urls.txt
webscanner -serve example.com              # scan, then show the result in your browser
webscanner view webscanner-example.com-20261015-101500   # show a saved result
```

Progress and a summary are shown while it runs. The results are saved to
`webscanner-<domain>-<time>/` (or `-out <folder>`):

| File | Contents |
| --- | --- |
| `result.json` | everything, in the same shape as the website's `/results` endpoint |
| `hosts.csv` | one row per host: state, sources, addresses, network, HTTP status, certificate |
| `urls.csv` | one row per URL: type, state, status, sources, first archived |
| `scan/` | the scan's database, which `webscanner view` reads |

`webscanner view <folder>` shows a saved result in your browser, with the
hosts, URLs, a site map, hosting networks and DNS records, like the website.
It is served from this computer only (127.0.0.1, a free port) and stops at
Ctrl-C. `-no-open` prints the address instead of opening the browser, and
`-addr` picks the address.

Press Ctrl-C once to stop a scan and keep what it found; twice to quit at
once.

| Flag | Meaning |
| --- | --- |
| `-mode` | `passive`, `standard` (default) or `full` |
| `-out` | folder for the result files |
| `-no-files` | write no files |
| `-serve` | show the result in your browser when the scan ends |
| `-print` | also print `hosts` or `urls` to standard output, one per line |
| `-pages`, `-depth` | full mode: pages per host (500) and link depth (3) |
| `-timeout` | stop after this long, keeping what was found (default: no limit) |
| `-quiet`, `-verbose` | no progress or summary; or a log of every step |
| `-no-memory` | do not remember subdomain sources' answers between runs |
| `-version` | print the version |

## Limits

What the website limits to share one small server is lifted here: there is
no queue, no bandwidth cap, no time limit and no ceiling on hosts or URLs.

What protects the scanned sites and the public sources stays, and cannot be
switched off: at most 5 requests per second to any one host, one web archive
request every 3 seconds (the Internet Archive blocks clients that ask
faster), and no guessing of pages or names, form submission, port scanning
or contact with private addresses.

Requests come from your computer and your IP address.

Subdomain sources' answers are remembered for 30 days in your user cache
folder (for example `~/.cache/webscanner` on Linux), so slow sources such as
crt.sh answer at once on the next scan.

## Releasing

Pushing a tag starting with `cli-v` builds the program for Linux, macOS and
Windows (Intel and ARM) and publishes a GitHub release with the files and a
`checksums.txt` ([release-cli.yml](../.github/workflows/release-cli.yml)):

```sh
git tag cli-v1.0.0
git push origin cli-v1.0.0
```

Each release also gets a Homebrew formula and a Scoop manifest
([release/manifests.sh](release/manifests.sh)). The formula is published to
the `homebrew-tap` repository when the `HOMEBREW_TAP_TOKEN` secret is set.

## Build

Needs Go 1.26 or newer. From this folder:

```sh
go build -o webscanner .
go test ./...
```

For other systems, set `GOOS` and `GOARCH`, for example
`GOOS=windows GOARCH=amd64 go build -o webscanner.exe .` or
`GOOS=darwin GOARCH=arm64 go build -o webscanner .`. The binary needs nothing
else installed. A release version is set with
`-ldflags "-X main.version=1.0.0"`.

The engines come from the server module (`../server`), which this module
uses through a `replace` line in `go.mod`, so the two always match.
