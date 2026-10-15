import type { Metadata } from "next";
import Link from "next/link";
import { Command } from "@/components/command";
import { JsonLd } from "@/components/json-ld";
import { sourceUrl } from "@/components/site-footer";
import { TerminalIcon } from "@/components/terminal-icon";

export const metadata: Metadata = {
  title: "Command-line tool: find subdomains and URLs from your terminal",
  description:
    "webscanner, a free command-line tool for Windows, macOS and Linux that lists a domain's subdomains and URLs from public sources and saves them as JSON and CSV. One-line install.",
  alternates: { canonical: "/cli" },
  openGraph: { url: "/cli", title: "webscanner command-line tool" },
};

// The installers and releases live in the GitHub repository.
const repo = sourceUrl.replace("https://github.com/", "");
const raw = `https://raw.githubusercontent.com/${repo}/main/cli`;
const releases = `${sourceUrl}/releases`;
const owner = repo.split("/")[0];

export default function CliPage() {
  return (
    <article className="mx-auto max-w-2xl space-y-10 text-sm leading-6">
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "SoftwareApplication",
          name: "webscanner",
          description: metadata.description,
          applicationCategory: "DeveloperApplication",
          operatingSystem: "Windows, macOS, Linux",
          downloadUrl: releases,
          softwareHelp: `${sourceUrl}/tree/main/cli`,
          license: "https://opensource.org/licenses/MIT",
          offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
        }}
      />
      <header className="space-y-3">
        <p className="flex items-center gap-2 text-muted">
          <TerminalIcon />
          webscanner
        </p>
        <h1 className="text-2xl font-semibold tracking-tight">Run Web Scanner on your computer</h1>
        <p className="text-muted">
          The command-line tool runs the same scan as this website, from your own machine. Without
          the website&apos;s limits, it lists every host and URL it finds, however many there are,
          and saves them as JSON and CSV files. It is free and needs nothing else installed.
        </p>
      </header>

      <Section title="Install">
        <h3 className="font-medium">Linux and macOS</h3>
        <Command label="install on Linux and macOS">{`curl -fsSL ${raw}/install.sh | sh`}</Command>
        <h3 className="pt-2 font-medium">Windows (PowerShell)</h3>
        <Command label="install on Windows">{`irm ${raw}/install.ps1 | iex`}</Command>
        <h3 className="pt-2 font-medium">With a package manager</h3>
        <Command label="install with Homebrew">{`brew install ${owner}/tap/webscanner`}</Command>
        <Command label="install with Scoop">{`scoop install ${releases}/latest/download/webscanner.json`}</Command>
        <p className="text-muted">
          The installer downloads the latest version for your system, checks it against the
          published checksums, and adds <span className="font-mono">webscanner</span> to your
          path. Prefer to download it yourself? Pick the file for your system on the{" "}
          <a href={releases} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline">
            releases page
          </a>
          . The program is not signed, so your system may warn you the first time you run a file
          downloaded with a browser.
        </p>
      </Section>

      <Section title="Scan">
        <Command label="standard scan">webscanner example.com</Command>
        <p className="text-muted">
          Shows progress while it runs, then a summary, and saves the results to a folder named
          after the domain. Press Ctrl-C to stop and keep what was found so far.
        </p>
        <h3 className="pt-2 font-medium">More examples</h3>
        <Command label="passive scan">webscanner -mode passive example.com</Command>
        <Command label="full crawl">webscanner -mode full -pages 1000 example.com</Command>
        <Command label="list URLs only">webscanner -print urls -no-files example.com &gt; urls.txt</Command>
        <Command label="show all options">webscanner -help</Command>
      </Section>

      <Section title="Browse a result">
        <Command label="scan, then open the result">webscanner -serve example.com</Command>
        <Command label="open a saved result">webscanner view webscanner-example.com-20261015-101500</Command>
        <p className="text-muted">
          Opens the result in your browser with hosts, URLs, a site map, hosting networks and DNS
          records, like the results on this website. The page is served from your own computer
          only and stops when you press Ctrl-C; nothing is uploaded.
        </p>
      </Section>

      <Section title="Modes">
        <Table
          rows={[
            ["passive", "Public sources only: certificate logs, subdomain databases, the web archive and DNS. Never contacts the site."],
            ["standard", "Also checks which hosts are live and reads their robots.txt and sitemaps. The default."],
            ["full", "Also follows links on each live host (-pages and -depth set how far)."],
          ]}
        />
      </Section>

      <Section title="What you get">
        <Table
          rows={[
            ["result.json", "Everything the scan found, with where each item came from."],
            ["hosts.csv", "One row per host: whether it resolves and answers, addresses, network, certificate."],
            ["urls.csv", "One row per URL: type, status, sources, when it was first archived."],
            ["scan/", "The scan's database, which webscanner view reads."],
          ]}
        />
      </Section>

      <Section title="Limits">
        <p>
          The website shares one small server, so it limits how much each scan may do. The
          command-line tool has none of those limits: no queue, no time limit, and no ceiling on
          hosts or URLs.
        </p>
        <p>
          What protects the sites you scan, and the public sources it reads, stays fixed: at most 5
          requests per second to any one host, one web archive request every 3 seconds, and no
          guessing of pages or names, form submission or port scanning.
        </p>
        <p className="text-muted">
          Requests come from your computer and your internet address. Only scan domains you own or
          have permission to inspect; the{" "}
          <Link href="/terms" className="text-accent hover:underline">
            Terms of Use
          </Link>{" "}
          apply.
        </p>
      </Section>

      <p className="flex flex-wrap gap-x-6 gap-y-2">
        <Link href="/" className="text-accent hover:underline">
          Scan on the website instead
        </Link>
        <a href={`${sourceUrl}/tree/main/cli`} target="_blank" rel="noopener noreferrer" className="text-accent hover:underline">
          Source code and full documentation
        </a>
      </p>
    </article>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-3">
      <h2 className="border-b border-border pb-1 text-lg font-semibold">{title}</h2>
      {children}
    </section>
  );
}

function Table({ rows }: { rows: [string, string][] }) {
  return (
    <dl className="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-x-4 gap-y-2 rounded-md border border-border bg-surface p-3">
      {rows.map(([term, text]) => (
        <div key={term} className="contents">
          <dt className="font-mono text-xs leading-6">{term}</dt>
          <dd className="text-muted">{text}</dd>
        </div>
      ))}
    </dl>
  );
}
