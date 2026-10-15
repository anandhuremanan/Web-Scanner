import Link from "next/link";
import { JsonLd } from "@/components/json-ld";
import { ScanForm } from "@/components/scan-form";
import { siteDescription, siteName, siteUrl } from "@/lib/site";
import { sourceUrl } from "@/components/site-footer";
import { TerminalIcon } from "@/components/terminal-icon";

export default function Home() {
  return (
    <div className="mx-auto max-w-xl pt-12">
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "WebApplication",
          name: siteName,
          url: siteUrl,
          description: siteDescription,
          applicationCategory: "SecurityApplication",
          operatingSystem: "Any",
          browserRequirements: "Requires JavaScript",
          isAccessibleForFree: true,
          offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
          featureList: [
            "Subdomains from certificate transparency logs and public subdomain databases",
            "URLs from the Wayback Machine, robots.txt and sitemaps",
            "Optional crawl of live hosts",
            "Hosting network, DNS records and TLS certificate of each host",
            "Source shown for every result; CSV export",
          ],
          sameAs: [sourceUrl],
        }}
      />
      <h1 className="text-3xl font-semibold tracking-tight">Web Scanner</h1>
      <p className="mt-2 text-muted">
        Understand what is publicly discoverable about your website.
      </p>

      <div className="mt-8">
        <ScanForm />
      </div>

      <Link
        href="/cli"
        className="mt-6 inline-flex items-center gap-2 rounded-md border border-border px-3 py-1.5 text-sm text-muted transition-colors hover:border-accent hover:text-fg"
      >
        <TerminalIcon />
        Run it on your computer with the command-line tool
      </Link>

      <section className="mt-12 space-y-3 text-sm leading-6 text-muted">
        <h2 className="font-medium text-fg">What the scan does</h2>
        <p>
          Every scan lists your domain&rsquo;s subdomains from public certificate logs and the
          routes a public web archive has recorded. Standard scans also check which subdomains
          are live and read their robots.txt and sitemaps; a full crawl additionally follows
          links on each live site. For each result the map shows how it was found.
        </p>
        <p>
          Discovery is low-impact: a passive scan never contacts your site, and the others make
          a limited number of ordinary GET requests at a modest rate. Nothing is submitted,
          guessed or brute-forced. Only scan domains you own or have permission to inspect.
        </p>
        <p>
          The result is a map of what the scanner could discover publicly — not a complete
          inventory of every route your server implements.
        </p>
      </section>
    </div>
  );
}
