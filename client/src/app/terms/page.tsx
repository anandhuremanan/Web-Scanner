import type { Metadata } from "next";
import Link from "next/link";
import { sourceUrl } from "@/components/site-footer";

export const metadata: Metadata = {
  title: "Terms of Use and Privacy · Web Scanner",
  description: "The rules for using Web Scanner, and what it stores about you.",
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
        Web Scanner lists what is publicly discoverable about a website: its subdomains and the
        pages on them, with the public source each one came from. By starting a scan you agree to
        these terms.
      </p>

      <h2 className="border-b border-border pb-1 text-lg font-semibold">Terms of Use</h2>

      <Section title="Who may scan what">
        <p>
          Only scan domains you own, or that you have permission from the owner to inspect. You are
          responsible for the scans you start and for how you use their results.
        </p>
      </Section>

      <Section title="What you must not do">
        <ul className="list-disc space-y-1 pl-5">
          <li>Use the service, or its results, for anything unlawful, or to harm or attack anyone.</li>
          <li>Scan systems you are not authorised to inspect.</li>
          <li>Try to get around its limits, overload it, or use it through automated scripts.</li>
        </ul>
      </Section>

      <Section title="What the service does">
        <p>
          A scan reads public sources: certificate transparency logs, public subdomain databases, a
          public web archive and DNS. A passive scan never contacts the scanned site. Standard and
          full scans make a limited number of ordinary page requests to it, at a modest rate. The
          service never logs in, submits forms, guesses pages or names, or tests for
          vulnerabilities.
        </p>
      </Section>

      <Section title="No guarantees">
        <p>
          The service is provided free, as is, without any warranty. Results can be incomplete,
          out of date or wrong: they show what public sources recorded, not a full inventory of a
          site. The service may be slow, unavailable, changed or stopped at any time.
        </p>
        <p>
          To the extent the law allows, the operator is not liable for any loss or damage arising
          from using the service or relying on its results.
        </p>
      </Section>

      <Section title="Our rights">
        <p>
          We may limit, refuse or stop scans, and remove results, at our discretion, for example to
          prevent misuse or to respect a site owner&apos;s request.
        </p>
      </Section>

      <h2 className="border-b border-border pb-1 text-lg font-semibold">Privacy</h2>

      <Section title="Scans and their results">
        <ul className="list-disc space-y-1 pl-5">
          <li>
            The domain you enter, the scan mode and the results are stored on our server for up to
            about a week, then deleted automatically (sooner if storage fills up).
          </li>
          <li>
            Anyone with the link to a result can open it while it is stored. Do not share the link
            if you do not want others to see the result.
          </li>
          <li>
            The server keeps short operational logs, which include the domains scanned. They are
            used to run and protect the service.
          </li>
          <li>
            To find results, the domain you enter is sent to the public sources listed above, and in
            standard and full scans the scanned site receives ordinary page requests from our server.
          </li>
        </ul>
      </Section>

      <Section title="Your connection">
        <ul className="list-disc space-y-1 pl-5">
          <li>
            Your IP address is used, in memory only, to limit how many scans one visitor can start.
            Our own logs do not record it.
          </li>
          <li>
            Before a scan, Cloudflare Turnstile checks that the visitor is a person, which involves
            your browser sending signals to Cloudflare.
          </li>
          <li>
            Our hosting providers (Vercel and Cloudflare) process your requests and may keep their
            standard logs.
          </li>
        </ul>
      </Section>

      <Section title="Usage statistics">
        <p>
          We count how the site is used with PostHog (hosted in the EU): scans started, finished or
          cancelled, tabs opened, and downloads. These statistics never include the domain you
          scan, its hosts or URLs. They use no cookies, store nothing in your browser and do not
          record your screen. PostHog receives your IP address with each event.
        </p>
      </Section>

      <Section title="Browser storage and cookies">
        <p>
          The site sets no cookies of its own. While a scan runs, it keeps an identifier for that
          scan in your browser tab (session storage), so you can cancel it; it is gone when you
          close the tab. Cloudflare Turnstile may use its own storage to do its check.
        </p>
      </Section>

      <Section title="Questions, removal and objections">
        <p>
          To ask a question, have a stored result removed, or object to a scan of your domain,{" "}
          <a
            href={`${sourceUrl}/issues`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent hover:underline"
          >
            open an issue on GitHub
          </a>
          . Please do not include personal details there; we will reply on the issue.
        </p>
      </Section>

      <Section title="Changes">
        <p>
          We may update these terms; the date at the top shows when. Using the service after a
          change means you accept the updated terms.
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
