# webscanner (command line)

Lists what is publicly discoverable about a domain, its subdomains and the
pages on them, and where each one was found, from your own computer. It runs
the same discovery engines as the Web Scanner website, without the website's
limits on how much one scan may do.

Only scan domains you own or have permission to inspect.

## Use

```sh
webscanner example.com                     # standard scan, results saved to a folder
webscanner -mode passive example.com       # never contacts the site
webscanner -mode full -pages 1000 example.com
webscanner -print urls -no-files example.com > urls.txt
```

Progress and a summary are shown while it runs. The results are saved to
`webscanner-<domain>-<time>/` (or `-out <folder>`):

| File | Contents |
| --- | --- |
| `result.json` | everything, in the same shape as the website's `/results` endpoint |
| `hosts.csv` | one row per host: state, sources, addresses, network, HTTP status, certificate |
| `urls.csv` | one row per URL: type, state, status, sources, first archived |

Press Ctrl-C once to stop a scan and keep what it found; twice to quit at
once.

| Flag | Meaning |
| --- | --- |
| `-mode` | `passive`, `standard` (default) or `full` |
| `-out` | folder for the result files |
| `-no-files` | write no files |
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
