package main

import (
	"log/slog"
	"net"
	"time"

	"websitemapper/internal/config"
	"websitemapper/internal/discovery"
	"websitemapper/internal/discovery/archive"
	"websitemapper/internal/discovery/dnsresolve"
	"websitemapper/internal/discovery/htmlcrawl"
	"websitemapper/internal/discovery/httpprobe"
	"websitemapper/internal/discovery/sitemap"
	"websitemapper/internal/discovery/subdomains"
	"websitemapper/internal/fetch"
	"websitemapper/internal/resource"
	"websitemapper/internal/scan"
)

// The stages are the server's (see server/cmd/server/main.go), wired for one
// scan on the user's own machine. What protects the server is lifted: there
// is no queue, no bandwidth cap and no ceiling on hosts or URLs. What
// protects the scanned sites and the public sources stays exactly as on the
// server: at most SCAN_REQUESTS_PER_SECOND requests per second to one host,
// one web archive request every 3 seconds, no guessing, no form submission
// and no private addresses.

const (
	// unlimited stands for "no limit" where an engine needs a number.
	unlimited = 1 << 30

	archivePageTimeout   = 2 * time.Minute
	fastSourceTimeout    = 10 * time.Second
	crtshDatabaseTimeout = 90 * time.Second
	// slowSourceWait is how long a scan with nothing else left to do waits
	// for crt.sh and Cert Spotter. Longer than on the server: someone running
	// the tool by hand would rather wait a minute than miss their answer.
	slowSourceWait = time.Minute
)

type pools struct {
	http, dns, ct, lookups, archive *resource.Pool
}

func newPools(cfg config.Config) pools {
	return pools{
		http:    resource.NewPool("http", cfg.GlobalHTTPConcurrency),
		dns:     resource.NewPool("dns", cfg.GlobalDNSConcurrency),
		ct:      resource.NewPool("certificate-transparency", 4),
		lookups: resource.NewPool("subdomain-lookup", 8),
		archive: resource.NewPool("archive", 1),
	}
}

func pipeline(cfg config.ScanConfig, p pools, names subdomains.Memory, log *slog.Logger) []scan.Stage {
	client := fetch.New(fetch.Options{
		Timeout:           cfg.RequestTimeout,
		ConnectTimeout:    cfg.ConnectTimeout,
		MaxBodyBytes:      cfg.MaxBodyBytes,
		UserAgent:         cfg.UserAgent,
		RequestsPerSecond: cfg.RequestsPerSecond,
		Pool:              p.http,
		ReadBody:          fetch.HTMLOnly,
	})

	dnsOptions := dnsresolve.Options{
		MaxHosts:    unlimited,
		Concurrency: p.dns.Stats().Capacity,
		Timeout:     cfg.DNSTimeout,
		Pool:        p.dns,
		NegativeTTL: time.Minute,
		Networks:    dnsresolve.NewNetworks(net.DefaultResolver),
		Records:     net.DefaultResolver,
	}
	dns := dnsresolve.New(net.DefaultResolver, dnsOptions)
	if len(cfg.DNSServers) > 0 {
		dns = dnsresolve.NewWith(&dnsresolve.Client{
			Servers: cfg.DNSServers, Timeout: 2 * time.Second, Fallback: net.DefaultResolver, Log: log,
		}, dnsOptions)
	}
	probe := httpprobe.New(client, httpprobe.Options{
		MaxHosts:    unlimited,
		Concurrency: max(cfg.HostConcurrency, p.http.Stats().Capacity),
	})
	crawler := htmlcrawl.New(client, htmlcrawl.Options{
		MaxHosts:        unlimited,
		MaxPagesPerHost: cfg.MaxURLs,
		MaxDepth:        cfg.MaxDepth,
		Concurrency:     cfg.Concurrency,
		HostConcurrency: cfg.HostConcurrency,
	}, log)
	sitemaps := sitemap.New(client.WithBodies(fetch.TextAndXML, 4<<20), sitemap.Options{
		MaxHosts:        unlimited,
		MaxFilesPerHost: 50,
		MaxURLsPerHost:  unlimited,
		Concurrency:     cfg.HostConcurrency,
	})

	all := []scan.Mode(nil)
	lightAndFull := []scan.Mode{scan.ModeLight, scan.ModeFull}
	archiveClient := fetch.New(fetch.Options{
		Timeout:      archivePageTimeout,
		MaxBodyBytes: 32 << 20,
		UserAgent:    cfg.UserAgent,
		// The Internet Archive's pace, kept as on the server: it blocks
		// clients that ask faster.
		RequestsPerSecond: 1.0 / 3,
		Pool:              p.archive,
	})
	stages := []scan.Stage{{ID: "archive", Label: "Searching web archives", Modes: all, Background: true,
		Engines: []discovery.Engine{archive.New(archiveClient, archive.Options{MaxURLs: unlimited, FailureTTL: time.Minute})}}}
	stages = append(stages, subdomainStages(cfg, p, names, log)...)
	return append(stages,
		scan.Stage{ID: "resolve", Label: "Resolving discovered hosts", Modes: all, Engines: []discovery.Engine{dns}},
		scan.Stage{ID: "probe", Label: "Checking which hosts are live", Modes: lightAndFull, Engines: []discovery.Engine{probe}},
		scan.Stage{ID: "sitemaps", Label: "Reading robots.txt and sitemaps", Modes: lightAndFull, Engines: []discovery.Engine{sitemaps}},
		scan.Stage{ID: "crawl", Label: "Crawling reachable hosts", Modes: []scan.Mode{scan.ModeFull}, Engines: []discovery.Engine{crawler}},
		scan.Stage{ID: "follow-up", Label: "Checking hosts found in the web archive", Modes: []scan.Mode{scan.ModePassive},
			Join: true, Engines: []discovery.Engine{dns}},
		scan.Stage{ID: "follow-up", Label: "Checking hosts found later", Modes: []scan.Mode{scan.ModeLight},
			Join: true, Engines: []discovery.Engine{dns, probe, sitemaps}},
		scan.Stage{ID: "follow-up", Label: "Checking hosts found later", Modes: []scan.Mode{scan.ModeFull},
			Join: true, Engines: []discovery.Engine{dns, probe, sitemaps, crawler}},
	)
}

func subdomainStages(cfg config.ScanConfig, p pools, names subdomains.Memory, log *slog.Logger) []scan.Stage {
	slowClient := fetch.New(fetch.Options{
		Timeout: cfg.CTTimeout, MaxBodyBytes: 64 << 20, UserAgent: cfg.UserAgent, RequestsPerSecond: 1, Pool: p.ct,
	})
	quickClient := fetch.New(fetch.Options{
		Timeout: fastSourceTimeout, MaxBodyBytes: 16 << 20, UserAgent: cfg.UserAgent, RequestsPerSecond: 1, Pool: p.lookups,
	})
	crtsh := subdomains.NewCRTSh(slowClient)
	crtsh.DB = &subdomains.CRTShDB{Pool: p.ct, Timeout: crtshDatabaseTimeout}
	slow := []subdomains.Source{crtsh, subdomains.NewCertSpotter(slowClient)}
	quick := []subdomains.Source{subdomains.NewAnubis(quickClient), subdomains.NewTHC(quickClient), subdomains.NewShodanCT(quickClient)}

	engine := func(sources []subdomains.Source, timeout time.Duration) []discovery.Engine {
		return []discovery.Engine{subdomains.New(sources, timeout, log).WithCache(subdomains.CacheOptions{
			TTL: 6 * time.Hour, RateLimitTTL: 15 * time.Minute, FailureTTL: 5 * time.Minute, Memory: names,
		})}
	}
	return []scan.Stage{
		{ID: "certificates", Label: "Searching certificate logs", Background: true, JoinWait: slowSourceWait,
			Engines: engine(slow, cfg.CTTimeout+crtshDatabaseTimeout)},
		{ID: "subdomains", Label: "Discovering subdomains", Engines: engine(quick, fastSourceTimeout)},
	}
}
