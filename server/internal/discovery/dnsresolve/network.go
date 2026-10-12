package dnsresolve

import (
	"context"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"websitemapper/internal/discovery"
)

// TXTResolver looks up TXT records. *net.Resolver satisfies it.
type TXTResolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// Networks finds which network an IP address belongs to: its number (ASN),
// operator and registered country. It asks Team Cymru's IP-to-ASN service,
// which answers over DNS and is meant for bulk use:
//
//	4.3.2.1.origin.asn.cymru.com   TXT  "64500 | 1.2.3.0/24 | US | arin | 2020-01-01"
//	AS64500.asn.cymru.com          TXT  "64500 | US | arin | 2010-01-01 | EXAMPLE-NET - Example Inc., US"
//
// Many hosts share few addresses, so answers are kept in memory.
type Networks struct {
	resolver TXTResolver
	// Timeout bounds one question (0: 3 s).
	Timeout time.Duration

	mu    sync.Mutex
	byIP  map[netip.Addr]discovery.Network
	names map[int]string
}

// maxNetworkEntries bounds the addresses remembered.
const maxNetworkEntries = 50000

// NewNetworks creates a Networks that asks through r.
func NewNetworks(r TXTResolver) *Networks {
	return &Networks{resolver: r, byIP: map[netip.Addr]discovery.Network{}, names: map[int]string{}}
}

// Lookup returns the network of ip. ok is false if it could not be found
// (a private address, an address nobody announces, or no answer in time).
func (n *Networks) Lookup(ctx context.Context, ip netip.Addr) (discovery.Network, bool) {
	ip = ip.Unmap()
	n.mu.Lock()
	net, ok := n.byIP[ip]
	n.mu.Unlock()
	if ok {
		return net, true
	}

	fields, ok := n.txt(ctx, originName(ip))
	if !ok || len(fields) < 3 {
		return discovery.Network{}, false
	}
	// An address announced by several networks lists them all; take the first.
	asn, err := strconv.Atoi(strings.Fields(fields[0] + " 0")[0])
	if err != nil || asn <= 0 {
		return discovery.Network{}, false
	}
	net = discovery.Network{ASN: asn, Prefix: fields[1], Country: fields[2], Name: n.name(ctx, asn)}

	// A network whose name could not be read is not remembered, so the name
	// is asked for again next time.
	if net.Name != "" {
		n.mu.Lock()
		if len(n.byIP) >= maxNetworkEntries {
			clear(n.byIP)
		}
		n.byIP[ip] = net
		n.mu.Unlock()
	}
	return net, true
}

// name returns the operator of a network, or "" if it could not be read.
func (n *Networks) name(ctx context.Context, asn int) string {
	n.mu.Lock()
	name, ok := n.names[asn]
	n.mu.Unlock()
	if ok {
		return name
	}
	fields, ok := n.txt(ctx, "AS"+strconv.Itoa(asn)+".asn.cymru.com")
	if !ok || len(fields) < 5 {
		return ""
	}
	name = operatorName(fields[4])
	n.mu.Lock()
	n.names[asn] = name
	n.mu.Unlock()
	return name
}

// txt asks one question, once more if the first try fails, and returns the
// "|"-separated fields of the first answer.
func (n *Networks) txt(ctx context.Context, name string) ([]string, bool) {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	for attempt := 0; attempt < 2 && ctx.Err() == nil; attempt++ {
		qctx, cancel := context.WithTimeout(ctx, timeout)
		answers, err := n.resolver.LookupTXT(qctx, name)
		cancel()
		if err != nil || len(answers) == 0 {
			continue
		}
		fields := strings.Split(answers[0], "|")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		return fields, true
	}
	return nil, false
}

// originName is the DNS name to ask for an address's network: the address
// reversed (by byte for IPv4, by hex digit for IPv6) under the service's zone.
func originName(ip netip.Addr) string {
	var b strings.Builder
	if ip.Is4() {
		a := ip.As4()
		for i := 3; i >= 0; i-- {
			b.WriteString(strconv.Itoa(int(a[i])))
			b.WriteByte('.')
		}
		return b.String() + "origin.asn.cymru.com"
	}
	const hex = "0123456789abcdef"
	a := ip.As16()
	for i := 15; i >= 0; i-- {
		b.WriteByte(hex[a[i]&0xf])
		b.WriteByte('.')
		b.WriteByte(hex[a[i]>>4])
		b.WriteByte('.')
	}
	return b.String() + "origin6.asn.cymru.com"
}

// operatorName makes a registry description readable: "CLOUDFLARENET -
// Cloudflare, Inc., US" becomes "Cloudflare, Inc."; "AS-HOSTINGER, CY"
// becomes "AS-HOSTINGER".
func operatorName(desc string) string {
	desc = strings.TrimSpace(desc)
	// A trailing ", CC" is the country, which is reported separately.
	if i := strings.LastIndex(desc, ", "); i > 0 && len(desc)-i == 4 {
		desc = desc[:i]
	}
	if _, after, ok := strings.Cut(desc, " - "); ok && strings.TrimSpace(after) != "" {
		desc = strings.TrimSpace(after)
	}
	return desc
}
