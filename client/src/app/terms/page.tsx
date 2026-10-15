import type { Metadata } from "next";
import Link from "next/link";
import { sourceUrl } from "@/components/site-footer";

export const metadata: Metadata = {
  title: "Terms of Use and Privacy",
  description: "The rules for using Web Scanner, and what it stores about you.",
  alternates: { canonical: "/terms" },
};

/** The date these terms last changed; update it with every change. */
const updated = "14 October 2026";

/**
 * Terms of Use and Privacy notice, on one page. Keep them true to what the
 * software does: if storage, logging or analytics change, change this too.
 */
export default function Terms() {
  return (
    <article className="mx-auto max-w-2xl space-y-8 text-sm leading-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Terms of Use and Privacy</h1>
        <p className="text-muted">Last updated {updated}</p>
      </header>

      <p>
        Web Scanner lists what public sources record about a website: its subdomains, their pages,
        and where each one was found. By starting a scan, you agree to these terms.
      </p>

      <h2 className="border-b border-border pb-1 text-lg font-semibold">Terms of Use</h2>

      <Section title="Permitted use">
        <p>
          Scan only domains you own or have the owner&rsquo;s permission to inspect. You are
          responsible for the scans you start and for how you use the results.
        </p>
      </Section>

      <Section title="Prohibited use">
        <p>Do not:</p>
        <ul className="list-disc space-y-1 pl-5">
          <li>use the service or its results unlawfully, or to harm or attack anyone;</li>
          <li>scan systems you are not authorised to inspect;</li>
          <li>circumvent its limits, overload it, or access it with automated scripts.</li>
        </ul>
      </Section>

      <Section title="How scans work">
        <p>
          Every scan queries public sources: certificate transparency logs, public subdomain
          databases, a public web archive, and DNS. Passive scans never contact the target site.
          Standard and full scans also send it a limited number of ordinary, rate-limited page
          requests. The service never logs in, submits forms, guesses URLs or hostnames, or tests
          for vulnerabilities.
        </p>
      </Section>

      <Section title="Third-party information">
        <p>
          Results are compiled from public third-party sources and from what target sites serve
          publicly. We do not control, verify or endorse that information, and listing a host or
          page says nothing about who owns or operates it.
        </p>
      </Section>

      <Section title="No warranty">
        <p>
          The service is free and provided &ldquo;as is&rdquo;, without warranty of any kind.
          Results reflect what public sources have recorded. They may be incomplete, outdated or
          wrong, and are not a full inventory of any site. We may change, suspend or discontinue the
          service at any time.
        </p>
      </Section>

      <Section title="Limitation of liability">
        <p>
          To the extent permitted by law, we are not liable for any loss or damage arising from
          your use of the service or reliance on its results.
        </p>
      </Section>

      <Section title="Indemnity">
        <p>
          You agree to indemnify us against claims, losses and costs arising from your scans, your
          use of the results, or your breach of these terms.
        </p>
      </Section>

      <Section title="Our rights">
        <p>
          We may limit, refuse or stop scans, and remove results, at our discretion. For example,
          we may do this to prevent misuse or at a site owner&rsquo;s request.
        </p>
      </Section>

      <Section title="Governing law">
        <p>
          These terms are governed by the laws of India. If any part of them is found
          unenforceable, the rest still applies.
        </p>
      </Section>

      <h2 className="border-b border-border pb-1 text-lg font-semibold">Privacy</h2>

      <Section title="What we store">
        <ul className="list-disc space-y-1 pl-5">
          <li>
            The domain, scan mode and results are kept on our server for up to 7 days, then deleted
            automatically (sooner if storage runs low).
          </li>
          <li>
            Anyone with a result&rsquo;s link can view it while it is stored. Share the link only
            with people you want to see it.
          </li>
          <li>
            Our server keeps brief operational logs, including scanned domains, to run and protect
            the service.
          </li>
          <li>
            The domain you enter is sent to the public sources above. In standard and full scans,
            the target site also receives page requests from our server.
          </li>
        </ul>
      </Section>

      <Section title="Your connection">
        <ul className="list-disc space-y-1 pl-5">
          <li>
            Your IP address is held in memory only, to limit scans per visitor. Our logs do not
            record it.
          </li>
          <li>
            Before a scan, Cloudflare Turnstile checks that you are human. To do this, your browser
            sends signals to Cloudflare.
          </li>
          <li>
            Our hosting providers, Vercel and Cloudflare, process your requests and may keep their
            standard logs.
          </li>
        </ul>
      </Section>

      <Section title="Usage statistics">
        <p>
          We use PostHog (EU-hosted) to count scans started, finished or cancelled, tabs opened,
          and downloads. These events never include the scanned domain, its hosts or URLs. PostHog
          sets no cookies, stores nothing in your browser and does not record your screen, but it
          receives your IP address with each event.
        </p>
      </Section>

      <Section title="Cookies and browser storage">
        <p>
          We set no cookies. While a scan runs, its ID is kept in your tab&rsquo;s session storage
          so you can cancel it; it is cleared when you close the tab. Cloudflare Turnstile may use
          its own storage for its check.
        </p>
      </Section>

      <Section title="Contact, removal and objections">
        <p>
          To ask a question, request removal of a stored result, or object to a scan of your
          domain,{" "}
          <a
            href={`${sourceUrl}/issues`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:underline"
          >
            open an issue on GitHub
          </a>
          . Do not post personal details there; we will reply on the issue.
        </p>
      </Section>

      <Section title="Changes">
        <p>
          We may update these terms; the date above shows the latest version. Continuing to use the
          service after an update means you accept it.
        </p>
      </Section>

      <p>
        <Link href="/" className="text-accent hover:underline">
          Back to Web Scanner
        </Link>
      </p>
    </article>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-2">
      <h3 className="font-medium">{title}</h3>
      {children}
    </section>
  );
}
