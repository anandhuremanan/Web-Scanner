package dnsresolve

import (
	"context"
	"net"
	"sort"
	"strings"
	"time"

	"websitemapper/internal/discovery"
)

// RecordResolver looks up a domain's mail, name-server and text records.
// *net.Resolver satisfies it.
type RecordResolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// Limits on what is kept from a domain's TXT records, which anyone can
// make arbitrarily many and long.
const (
	maxTXTRecords = 40
	maxTXTLength  = 600
)

// domainRecords reads the public DNS records that describe a domain as a
// whole: three questions, asked once per scan. Records that cannot be read
// are left out; nil means none could.
func domainRecords(ctx context.Context, r RecordResolver, domain string) *discovery.DomainRecords {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	out := &discovery.DomainRecords{}

	if mx, err := r.LookupMX(ctx, domain); err == nil {
		for _, m := range mx {
			host := strings.TrimSuffix(strings.ToLower(m.Host), ".")
			if host != "" { // "." means the domain accepts no mail
				out.MX = append(out.MX, discovery.MXRecord{Host: host, Priority: int(m.Pref)})
			}
		}
		sort.SliceStable(out.MX, func(i, j int) bool { return out.MX[i].Priority < out.MX[j].Priority })
	}
	if ns, err := r.LookupNS(ctx, domain); err == nil {
		for _, n := range ns {
			out.NS = append(out.NS, strings.TrimSuffix(strings.ToLower(n.Host), "."))
		}
		sort.Strings(out.NS)
	}
	if txt, err := r.LookupTXT(ctx, domain); err == nil {
		sort.Strings(txt)
		for _, t := range txt {
			if len(out.TXT) == maxTXTRecords {
				break
			}
			if len(t) > maxTXTLength {
				t = t[:maxTXTLength] + "..."
			}
			out.TXT = append(out.TXT, t)
		}
	}
	if len(out.MX) == 0 && len(out.NS) == 0 && len(out.TXT) == 0 {
		return nil
	}
	out.Services = services(out)
	return out
}

// A serviceRule names a service that a record reveals the domain uses.
type serviceRule struct {
	name string
	// contains is looked for in the record, lowercased.
	contains string
}

// Rules are deliberately simple and evidence-based, like the technology
// rules: the evidence shown is the record itself.
var (
	mxRules = []serviceRule{
		{"Google Workspace (mail)", "google.com"},
		{"Google Workspace (mail)", "googlemail.com"},
		{"Microsoft 365 (mail)", "outlook.com"},
		{"Zoho Mail", "zoho."},
		{"Proton Mail", "protonmail.ch"},
		{"Fastmail", "messagingengine.com"},
		{"Mimecast (mail filtering)", "mimecast.com"},
		{"Proofpoint (mail filtering)", "pphosted.com"},
		{"Amazon SES (mail)", "amazonaws.com"},
		{"Cloudflare Email Routing", "mx.cloudflare.net"},
		{"Mailgun", "mailgun.org"},
	}
	nsRules = []serviceRule{
		{"Cloudflare (DNS)", ".ns.cloudflare.com"},
		{"Amazon Route 53 (DNS)", "awsdns-"},
		{"Google Cloud DNS", "googledomains.com"},
		{"Google Cloud DNS", "ns-cloud-"},
		{"Azure DNS", "azure-dns."},
		{"GoDaddy (DNS)", "domaincontrol.com"},
		{"Namecheap (DNS)", "registrar-servers.com"},
		{"DigitalOcean (DNS)", "digitalocean.com"},
		{"Vercel (DNS)", "vercel-dns.com"},
		{"Hostinger (DNS)", "dns-parking.com"},
	}
	txtRules = []serviceRule{
		{"Google (site verification)", "google-site-verification="},
		{"Microsoft 365 (domain verification)", "ms="},
		{"Facebook (domain verification)", "facebook-domain-verification="},
		{"Apple (domain verification)", "apple-domain-verification="},
		{"Atlassian (domain verification)", "atlassian-domain-verification="},
		{"Zoho (domain verification)", "zoho-verification="},
		{"Stripe (domain verification)", "stripe-verification="},
		{"OpenAI (domain verification)", "openai-domain-verification="},
		{"Google Workspace (sends mail)", "include:_spf.google.com"},
		{"Microsoft 365 (sends mail)", "include:spf.protection.outlook.com"},
		{"Zoho Mail (sends mail)", "include:zoho"},
		{"SendGrid (sends mail)", "include:sendgrid.net"},
		{"Mailgun (sends mail)", "include:mailgun.org"},
		{"Amazon SES (sends mail)", "include:amazonses.com"},
		{"Mailchimp (sends mail)", "include:servers.mcsv.net"},
	}
)

// services lists the services the records reveal, each with the record
// that reveals it, sorted by name.
func services(r *discovery.DomainRecords) []discovery.Service {
	found := map[string]string{}
	match := func(rules []serviceRule, kind, record string) {
		lower := strings.ToLower(record)
		for _, rule := range rules {
			hit := strings.Contains(lower, rule.contains)
			if rule.contains == "ms=" { // only as the whole record, "MS=ms12345"
				hit = strings.HasPrefix(lower, "ms=")
			}
			if _, had := found[rule.name]; hit && !had {
				found[rule.name] = kind + " " + record
			}
		}
	}
	for _, m := range r.MX {
		match(mxRules, "MX", m.Host)
	}
	for _, n := range r.NS {
		match(nsRules, "NS", n)
	}
	for _, t := range r.TXT {
		match(txtRules, "TXT", t)
	}
	out := make([]discovery.Service, 0, len(found))
	for name, ev := range found {
		out = append(out, discovery.Service{Name: name, Evidence: ev})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
