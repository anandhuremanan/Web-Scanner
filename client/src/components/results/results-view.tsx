"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { track } from "@/lib/analytics";
import { createScan, exportUrl, getHosts } from "@/lib/api";
import {
  formatDuration,
  hostStatus,
  modeLabels,
  networkLabel,
  plural,
  sourceLabel,
} from "@/lib/format";
import type { DomainRecords, Provider, Scan, ScanSummary } from "@/lib/types";
import { usePages } from "@/lib/use-pages";
import { useVerification } from "@/lib/verification";
import { HostList } from "./host-list";
import { SiteMap } from "./site-map";
import { ListFooter, UrlList } from "./url-list";

type Tab = "overview" | "hosts" | "hosting" | "dns" | "map" | "pages" | "apis" | "assets" | "technologies";

/** What a running scan has found so far, in the shape of a result summary. */
export function liveSummary(scan: Scan): ScanSummary {
  return {
    scanId: scan.id,
    status: scan.status,
    limits: scan.limits,
    domain: {
      mode: scan.mode,
      target: scan.target,
      canonical: scan.domain,
      startUrl: scan.startUrl,
      scannedAt: scan.startedAt ?? scan.createdAt,
      durationMs: 0,
    },
    errors: scan.errors,
    counts: scan.counts,
    technologies: [],
  };
}

interface Props {
  scan: Scan;
  result: ScanSummary;
  /**
   * The scan is still running: `result` is what it has found so far (see
   * liveSummary). Counts follow the scan; lists show what had been found
   * when they were opened, until they are refreshed.
   */
  live?: boolean;
}

/**
 * A scan's result. Only the summary is loaded up front; each tab reads its
 * hosts and URLs from the server in pages as they are shown.
 */
export function ResultsView({ scan, result, live = false }: Props) {
  const [tab, setTabState] = useState<Tab>("overview");
  const [openHost, setOpenHost] = useState<string | null>(null);
  // Changing this starts every list again from its first page.
  const [refresh, setRefresh] = useState(0);
  const setTab = (next: Tab) => {
    if (next !== tab) track("result_tab_opened", { tab: next, live });
    setTabState(next);
  };

  const { counts, domain } = result;
  const providers = result.providers ?? [];
  // The Pages tab also lists URLs of unknown type (for example form targets).
  const pages = counts.urls - counts.apis - counts.assets;

  const tabs: { id: Tab; label: string; count?: number }[] = [
    { id: "overview", label: "Overview" },
    { id: "hosts", label: "Hosts", count: counts.hosts },
    { id: "map", label: "Map" },
    { id: "pages", label: "Pages", count: pages },
    { id: "apis", label: "APIs", count: counts.apis },
    { id: "assets", label: "Assets", count: counts.assets },
    // Technologies and hosting are worked out from the whole result, once
    // it is complete.
    ...(live
      ? []
      : [
          { id: "hosting" as const, label: "Hosting", count: providers.length },
          { id: "dns" as const, label: "DNS" },
          { id: "technologies" as const, label: "Technologies", count: result.technologies.length },
        ]),
  ];

  const showHost = (hostname: string) => {
    setOpenHost(hostname);
    setTab("hosts");
  };

  return (
    <div className="space-y-6">
      {live ? (
        <header className="space-y-1">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h2 className="text-lg font-semibold">Found so far</h2>
            <button
              type="button"
              onClick={() => setRefresh((n) => n + 1)}
              className="text-sm text-accent hover:underline"
            >
              Refresh lists
            </button>
          </div>
          <p className="text-xs text-muted">
            The counts follow the scan. Lists show what had been found when they were opened.
          </p>
        </header>
      ) : (
        <header className="space-y-2">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h1 className="font-mono text-2xl font-semibold tracking-tight">{domain.canonical}</h1>
            <span className="flex flex-wrap items-baseline gap-x-4 text-sm">
              {counts.urls > 0 && (
                <a
                  href={exportUrl(scan.id)}
                  download
                  onClick={() => track("urls_exported", { urls: counts.urls })}
                  className="text-accent hover:underline"
                >
                  Download URLs (CSV)
                </a>
              )}
              <ShareButtons />
              <ScanAgain scan={scan} />
              <Link href="/" className="text-accent hover:underline print:hidden">
                New scan
              </Link>
            </span>
          </div>
          <p className="text-sm">
            {plural(counts.hosts, "host")} ({counts.hostsReachable} reachable) ·{" "}
            {plural(counts.urls, "URL")} · {plural(counts.apis, "API-like endpoint")} ·{" "}
            {plural(counts.assets, "asset")}
          </p>
          <p className="text-xs text-muted">
            {modeLabels[domain.mode] ?? domain.mode} scan from{" "}
            <span className="font-mono">{domain.startUrl}</span> ·{" "}
            {new Date(domain.scannedAt).toLocaleString()} · took{" "}
            {formatDuration(domain.durationMs)}
          </p>
        </header>
      )}

      {scan.status === "failed" && (
        <p className="rounded-md border border-border bg-surface px-3 py-2 text-sm">
          <span className="font-medium text-danger">The scan could not complete.</span>{" "}
          {scan.error} Any partial results are shown below.
        </p>
      )}

      {!live && (scan.status === "cancelled" || result.limits.length > 0) && (
        <div className="space-y-1 rounded-md border border-border bg-surface px-3 py-2 text-sm">
          {scan.status === "cancelled" && (
            <p className="font-medium">
              {scan.stopReason === "server_shutdown"
                ? "The scan was stopped because the server shut down."
                : "The scan was cancelled."}{" "}
              Results collected before it stopped are shown below.
            </p>
          )}
          {result.limits.map((l, i) => (
            <p key={i} className="text-muted">
              {l.message}
            </p>
          ))}
        </div>
      )}

      <nav className="-mx-4 overflow-x-auto px-4 print:hidden" aria-label="Result sections">
        <div className="flex gap-1 border-b border-border">
          {tabs.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => setTab(t.id)}
              aria-current={tab === t.id ? "page" : undefined}
              className={`-mb-px whitespace-nowrap border-b-2 px-3 py-2 text-sm ${
                tab === t.id
                  ? "border-fg font-medium"
                  : "border-transparent text-muted hover:text-fg"
              }`}
            >
              {t.label}
              {t.count !== undefined && (
                <span className="ml-1.5 font-mono text-xs text-muted">
                  {t.count.toLocaleString()}
                </span>
              )}
            </button>
          ))}
        </div>
      </nav>

      {/* Lists start again when refreshed, and when the scan finishes. */}
      <section key={`${refresh}|${live}`}>
        {tab === "overview" && (
          <Overview
            scanId={scan.id}
            result={result}
            live={live}
            goTo={setTab}
            showHost={showHost}
          />
        )}
        {tab === "hosts" && (
          <HostList
            key={openHost ?? ""}
            scanId={scan.id}
            apex={domain.canonical}
            total={counts.hosts}
            initialOpen={openHost}
          />
        )}
        {tab === "map" && <SiteMap scanId={scan.id} />}
        {tab === "pages" && (
          <UrlList
            scanId={scan.id}
            types={["page", "unknown"]}
            total={pages}
            showKind
            empty="No pages were discovered."
            note="Pages are URLs that returned HTML or were linked from HTML, on every host. Unknown URLs (for example form targets) are listed here too."
          />
        )}
        {tab === "apis" && (
          <UrlList
            scanId={scan.id}
            types={["api"]}
            total={counts.apis}
            empty="No API-like endpoints were discovered."
            note="API-like means the URL returned JSON, or its path looks like an API route. Open an entry to see which evidence applies — a matching path alone is not proof."
          />
        )}
        {tab === "assets" && (
          <UrlList
            scanId={scan.id}
            types={["asset"]}
            total={counts.assets}
            showKind
            empty="No assets were discovered."
          />
        )}
        {tab === "hosting" && (
          <Hosting scanId={scan.id} apex={domain.canonical} providers={providers} />
        )}
        {tab === "dns" && <DnsRecords records={result.dns} domain={domain.canonical} />}
        {tab === "technologies" && <Technologies result={result} />}
      </section>
    </div>
  );
}

/** Starts a new scan of the same target and mode, ignoring recent results. */
function ScanAgain({ scan }: { scan: Scan }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Checked only when the button is pressed, not on every result page.
  const { attach: attachVerification, getToken } = useVerification({ onDemand: true });

  async function onClick() {
    setBusy(true);
    setError(null);
    try {
      const verificationToken = await getToken();
      const next = await createScan(scan.target, scan.mode, { fresh: true, verificationToken });
      track("scan_again", { mode: scan.mode });
      router.push(`/scans/${next.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={onClick}
        disabled={busy}
        className="text-accent hover:underline disabled:opacity-50"
      >
        {busy ? "Starting…" : "Scan again"}
      </button>
      <div ref={attachVerification} />
      {error && (
        <span role="alert" className="text-warn">
          {error}
        </span>
      )}
    </>
  );
}

interface OverviewProps {
  scanId: string;
  result: ScanSummary;
  live: boolean;
  goTo: (t: Tab) => void;
  showHost: (hostname: string) => void;
}

function Overview({ scanId, result, live, goTo, showHost }: OverviewProps) {
  const { counts } = result;
  const rows: { label: string; value: number; tab: Tab }[] = [
    { label: "Hosts discovered", value: counts.hosts, tab: "hosts" },
    { label: "Hosts that resolve", value: counts.hostsResolved, tab: "hosts" },
    { label: "Hosts answering HTTP(S)", value: counts.hostsReachable, tab: "hosts" },
    { label: "Hosts crawled", value: counts.hostsCrawled, tab: "hosts" },
    { label: "Pages", value: counts.pages, tab: "pages" },
    { label: "API-like endpoints", value: counts.apis, tab: "apis" },
    { label: "Assets", value: counts.assets, tab: "assets" },
    // Technologies are detected once the scan is complete.
    ...(live
      ? []
      : [{ label: "Technologies", value: result.technologies.length, tab: "technologies" as const }]),
  ];

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-[18rem_minmax(0,1fr)]">
      <div className="space-y-6">
        <table className="w-full text-sm">
          <tbody className="divide-y divide-border">
            {rows.map((r) => (
              <tr key={r.label}>
                <td className="py-2">
                  <button type="button" onClick={() => goTo(r.tab)} className="hover:underline">
                    {r.label}
                  </button>
                </td>
                <td className="py-2 text-right font-mono tabular-nums">
                  {r.value.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <p className="text-sm leading-6 text-muted">
          This map shows what the scanner could discover publicly: hostnames from certificate
          logs and links, and routes linked from pages it crawled. It is not a complete
          inventory of DNS records or server-side routes.
        </p>

        {result.danglingAliases && result.danglingAliases.length > 0 && (
          <div className="text-sm">
            <h2 className="mb-2 font-medium">Aliases to names that do not exist</h2>
            <p className="mb-2 text-muted">
              These hosts are aliases (CNAME) for a name that no longer exists. If the missing name is
              at an outside service where anyone can register it, someone else could claim it. This
              was not checked; open a host to see where it points.
            </p>
            <ul className="space-y-1">
              {result.danglingAliases.map((h) => (
                <li key={h}>
                  <button
                    type="button"
                    onClick={() => showHost(h)}
                    className="break-all font-mono text-accent hover:underline"
                  >
                    {h}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}

        {result.errors.length > 0 && (
          <div className="text-sm">
            <h2 className="mb-2 font-medium">Problems during the scan</h2>
            <ul className="space-y-1">
              {result.errors.map((e, i) => (
                <li key={i}>
                  <span className="font-mono">{sourceLabel(e.engine)}</span>
                  {e.partial && <span className="text-muted"> (partly succeeded)</span>}
                  <span className="text-muted"> — {e.message}</span>
                </li>
              ))}
            </ul>
            <p className="mt-2 text-muted">Results from the other steps are still shown.</p>
          </div>
        )}
      </div>

      <OverviewHosts scanId={scanId} showHost={showHost} />
    </div>
  );
}

/** The result's hosts with their status, a page at a time. */
function OverviewHosts({ scanId, showHost }: Pick<OverviewProps, "scanId" | "showHost">) {
  const { pages, loading, error, hasMore, loadMore } = usePages((after) =>
    getHosts(scanId, { after }),
  );
  const hosts = pages.flatMap((p) => p.hosts);

  return (
    <div className="space-y-3 text-sm">
      <h2 className="font-medium">Hosts</h2>
      <ul className="divide-y divide-border rounded-md border border-border bg-surface">
        {hosts.map((h) => {
          const st = hostStatus(h);
          return (
            <li key={h.hostname}>
              <button
                type="button"
                onClick={() => showHost(h.hostname)}
                className="grid w-full grid-cols-[1.25rem_minmax(0,1fr)_auto] items-center gap-x-2 px-3 py-1.5 text-left hover:bg-subtle"
              >
                <span aria-hidden className={`text-center ${st.tone}`}>
                  {st.icon}
                </span>
                <span className="truncate font-mono">{h.hostname}</span>
                <span className={`text-right font-mono text-xs ${st.tone}`}>
                  {st.label}
                  {h.counts.urls > 0 && (
                    <span className="ml-3 text-muted">{plural(h.counts.urls, "URL")}</span>
                  )}
                </span>
              </button>
            </li>
          );
        })}
        {hosts.length === 0 && (
          <li className="px-3 py-3 text-muted">
            {loading ? "Loading…" : error ? "" : "No hosts were discovered."}
          </li>
        )}
      </ul>
      <ListFooter
        shown={hosts.length}
        loading={loading}
        error={error}
        hasMore={hasMore}
        onMore={loadMore}
      />
    </div>
  );
}

/** Copies the result's link, or prints the page (which can save a PDF). */
function ShareButtons() {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      track("result_shared", { how: "link" });
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // No clipboard access (an old browser, or not a secure page): the
      // address bar still has the link.
      window.prompt("Copy this link:", window.location.href);
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={copy}
        title="Anyone with the link can open this result while it is stored (about a week)"
        className="text-accent hover:underline print:hidden"
      >
        {copied ? "Link copied" : "Copy link"}
      </button>
      <button
        type="button"
        onClick={() => {
          track("result_shared", { how: "print" });
          window.print();
        }}
        title="Prints what is shown on this tab; choose 'Save as PDF' to keep a file"
        className="text-accent hover:underline print:hidden"
      >
        Print / PDF
      </button>
    </>
  );
}

/** The domain's own DNS records and what they reveal. */
function DnsRecords({ records, domain }: { records?: DomainRecords; domain: string }) {
  if (!records) {
    return (
      <p className="text-sm text-muted">
        No DNS records were read for {domain}: the lookups failed, or this result was saved before
        records were collected.
      </p>
    );
  }
  const { mx = [], ns = [], txt = [], services = [] } = records;
  return (
    <div className="space-y-6 text-sm">
      <p className="text-muted">
        The public DNS records of <span className="font-mono">{domain}</span> itself. Each
        host&apos;s own addresses are under Hosts.
      </p>

      {services.length > 0 && (
        <RecordSection title="Services these records reveal">
          <ul className="divide-y divide-border rounded-md border border-border bg-surface">
            {services.map((s) => (
              <li key={s.name} className="px-3 py-2">
                <div className="font-medium">{s.name}</div>
                <div className="break-all font-mono text-xs text-muted">{s.evidence}</div>
              </li>
            ))}
          </ul>
        </RecordSection>
      )}

      <RecordSection title="Mail servers (MX)" empty={mx.length === 0 && "None: no mail servers are published."}>
        <RecordList items={mx.map((m) => `${m.priority}  ${m.host}`)} />
      </RecordSection>
      <RecordSection title="Name servers (NS)" empty={ns.length === 0 && "None were returned."}>
        <RecordList items={ns} />
      </RecordSection>
      <RecordSection title="Text records (TXT)" empty={txt.length === 0 && "None."}>
        <RecordList items={txt} />
      </RecordSection>
    </div>
  );
}

function RecordSection({
  title,
  empty,
  children,
}: {
  title: string;
  empty?: string | false;
  children: React.ReactNode;
}) {
  return (
    <div>
      <h2 className="mb-2 font-medium">{title}</h2>
      {empty ? <p className="text-muted">{empty}</p> : children}
    </div>
  );
}

function RecordList({ items }: { items: string[] }) {
  return (
    <ul className="divide-y divide-border rounded-md border border-border bg-surface">
      {items.map((item) => (
        <li key={item} className="whitespace-pre-wrap break-all px-3 py-1.5 font-mono text-xs">
          {item}
        </li>
      ))}
    </ul>
  );
}

interface HostingProps {
  scanId: string;
  apex: string;
  providers: Provider[];
}

/**
 * Where the hosts are served from: the networks their addresses belong to,
 * largest first. Opening one lists its hosts.
 */
function Hosting({ scanId, apex, providers }: HostingProps) {
  const [open, setOpen] = useState<number | null>(null);
  if (providers.length === 0) {
    return (
      <p className="text-sm text-muted">
        No hosting information: none of the hosts resolved to a public address, or this result was
        saved before hosting was recorded.
      </p>
    );
  }
  const most = providers[0].hosts;
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted">
        The network each host&apos;s address belongs to. A host behind a CDN or proxy such as
        Cloudflare shows that service, not the server behind it. The country is where the address
        block is registered, which is not always where the server is.
      </p>
      <ul className="divide-y divide-border rounded-md border border-border bg-surface">
        {providers.map((p) => {
          const isOpen = open === p.asn;
          return (
            <li key={p.asn}>
              <button
                type="button"
                onClick={() => setOpen(isOpen ? null : p.asn)}
                aria-expanded={isOpen}
                className="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 px-3 py-2 text-left text-sm hover:bg-subtle"
              >
                <span className="min-w-0">
                  <span className="break-words font-medium">{networkLabel(p)}</span>
                  {p.country && <span className="ml-2 text-xs text-muted">{p.country}</span>}
                </span>
                <span className="font-mono text-xs text-muted">{plural(p.hosts, "host")}</span>
                <span aria-hidden className="col-span-2 h-1 overflow-hidden rounded bg-subtle">
                  <span
                    className="block h-full bg-accent"
                    style={{ width: `${Math.max((100 * p.hosts) / most, 2)}%` }}
                  />
                </span>
              </button>
              {isOpen && (
                <div className="border-t border-border bg-subtle/60 px-3 py-4">
                  <HostList scanId={scanId} apex={apex} total={p.hosts} asn={p.asn} />
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function Technologies({ result }: { result: ScanSummary }) {
  if (result.technologies.length === 0) {
    return (
      <p className="text-sm text-muted">
        No technologies were identified from the evidence collected.
      </p>
    );
  }
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted">
        Detected from simple, visible evidence such as response headers and URL patterns.
      </p>
      <ul className="divide-y divide-border rounded-md border border-border bg-surface">
        {result.technologies.map((t) => (
          <li key={t.name} className="px-3 py-3 text-sm">
            <div className="font-medium">{t.name}</div>
            <ul className="mt-1 space-y-0.5">
              {t.evidence.map((ev) => (
                <li key={ev} className="break-all font-mono text-xs text-muted">
                  {ev}
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </div>
  );
}
