import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "About Web Scanner's requests",
  description:
    "What the WebScanner user agent in your server logs is, what it requests, who starts it, and how to have your domain excluded.",
  alternates: { canonical: "/bot" },
};

/** What the scanner calls itself; keep it in step with the server's default. */
const userAgent = "WebScanner/1.0 (+https://webscanner.imanandhu.in/bot)";

/** Where site owners reach us to opt out, object or ask a question. */
const contactUrl = "https://www.pflabs.in/contact";

/**
 * The page the scanner's user agent links to, for the people who run the
 * sites it visits. Keep it true to what the scanner does.
 */
export default function Bot() {
  return (
    <article className="mx-auto max-w-2xl space-y-8 text-sm leading-6">
      <header className="space-y-3">
        <h1 className="text-2xl font-semibold tracking-tight">About Web Scanner&rsquo;s requests</h1>
        <p>If you found this user agent in your server&rsquo;s logs:</p>
        <p className="break-all rounded-md border border-border bg-surface px-3 py-2 font-mono text-xs">
          {userAgent}
        </p>
        <p>
          those requests came from{" "}
          <Link href="/" className="text-accent hover:underline">
            Web Scanner
          </Link>
          , a tool that shows what is publicly discoverable about a website: its subdomains, their
          pages, and where each was found. It is built for site owners and their security teams to
          review their own public footprint.
        </p>
      </header>

      <Section title="Who started it">
        <p>
          A person starts every scan; there are no automatic or scheduled scans. Before each scan
          they confirm that they own the domain or have explicit permission from its owner to
          inspect it, and agree to the{" "}
          <Link href="/terms" className="text-accent hover:underline">
            Terms of Use
          </Link>
          . We record that confirmation with each scan.
        </p>
      </Section>

      <Section title="What it requests">
        <ul className="list-disc space-y-1 pl-5">
          <li>
            <strong>Passive scans</strong> send nothing to your servers. They read only public
            sources: certificate transparency logs, public subdomain databases, a public web
            archive, and DNS through public resolvers.
          </li>
          <li>
            <strong>Standard scans</strong> also send a few requests to each of your subdomains
            that answers: its front page, its <code>robots.txt</code>, and the sitemaps listed
            there.
          </li>
          <li>
            <strong>Full crawls</strong> also follow the links on those pages, within your domain.
          </li>
        </ul>
        <p>
          Every request is an ordinary <code>GET</code>, rate-limited per host and capped per scan.
          The scanner never logs in, submits forms, sends data, guesses addresses or hostnames, or
          tests for vulnerabilities. It does not try to reach private or internal addresses.
        </p>
        <p>
          It reads <code>robots.txt</code> only for the sitemaps it lists, and does not treat its
          other rules as an opt-out. To stop scans of your domain, contact us as described below.
        </p>
      </Section>

      <Section title="Requests from other places">
        <p>
          Web Scanner is open source, and its{" "}
          <Link href="/cli" className="text-accent hover:underline">
            command-line tool
          </Link>{" "}
          sends the same user agent. Requests from it come from the computer of whoever runs it, not
          from us; we do not see or store them.
        </p>
      </Section>

      <Section title="Excluding your domain">
        <p>
          To have your domain and its subdomains excluded from future scans on this site, or to have
          a stored result removed sooner, tell us through our{" "}
          <a
            href={contactUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:underline"
          >
            contact page
          </a>
          . Say which domain, and that you are its owner or act for them. Stored results are
          deleted automatically after 7 days in any case.
        </p>
      </Section>
    </article>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-2">
      <h2 className="text-base font-semibold">{title}</h2>
      {children}
    </section>
  );
}
