// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Counts, Scan, ScanSummary } from "@/lib/types";

const api = vi.hoisted(() => ({
  createScan: vi.fn(),
  getHosts: vi.fn(),
  getUrls: vi.fn(),
  getTree: vi.fn(),
  exportUrl: (id: string) => `/api/scans/${id}/export`,
}));
vi.mock("@/lib/api", () => api);
const push = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

import { liveSummary, ResultsView } from "./results-view";

afterEach(cleanup);

/** A tab of the result sections, by the start of its label. */
function tab(name: RegExp) {
  return within(screen.getByRole("navigation")).queryByRole("button", { name });
}

const counts: Counts = {
  hosts: 2,
  hostsResolved: 2,
  hostsReachable: 1,
  hostsCrawled: 0,
  hostsUnresolved: 0,
  hostsResolvePending: 0,
  hostsProbed: 1,
  hostsUnreachable: 0,
  hostsProbePending: 0,
  hostsCrawlPending: 0,
  urls: 10,
  pages: 6,
  apis: 1,
  assets: 2,
  javascript: 1,
  urlsFetched: 0,
  urlsFailed: 0,
  cache: { hosts: 0, dns: 0, probes: 0, pages: 0 },
  limits: {
    hostsOmitted: 0,
    resolveSkipped: 0,
    probeSkipped: 0,
    crawlSkipped: 0,
    crawlLimited: 0,
    urlsOmitted: 0,
  },
};

function scan(status: Scan["status"]): Scan {
  return {
    id: "s1",
    target: "example.com",
    domain: "example.com",
    startUrl: "https://example.com/",
    mode: "light",
    status,
    createdAt: "2026-10-01T10:00:00Z",
    startedAt: "2026-10-01T10:00:01Z",
    phase: status === "running" ? "probing_hosts" : "done",
    limits: [],
    resources: { requests: 0 },
    steps: [],
    counts,
    errors: [],
  };
}

const finished: ScanSummary = {
  ...liveSummary(scan("completed")),
  technologies: [{ name: "nginx", evidence: ["Server: nginx"] }],
};

beforeEach(() => {
  push.mockReset();
  api.createScan.mockReset().mockResolvedValue({ id: "s2" });
  api.getHosts.mockReset().mockResolvedValue({
    hosts: [
      {
        hostname: "example.com",
        state: "reachable",
        sources: ["target"],
        counts: { urls: 10, pages: 6, apis: 1, assets: 2 },
      },
    ],
  });
});

describe("ResultsView for a finished scan", () => {
  it("counts the tabs from the summary, including unknown URLs as pages", async () => {
    render(<ResultsView scan={scan("completed")} result={finished} />);
    await screen.findByText("example.com", { selector: "span" });
    // 10 URLs - 1 API - 2 assets = 7 in the Pages tab.
    expect(tab(/^Pages/)?.textContent).toBe("Pages7");
    expect(tab(/^Technologies/)?.textContent).toBe("Technologies1");
    expect(screen.getByRole("link", { name: "Download URLs (CSV)" }).getAttribute("href")).toBe(
      "/api/scans/s1/export",
    );
  });

  it("starts a fresh scan of the same target and mode on request", async () => {
    render(<ResultsView scan={scan("completed")} result={finished} />);
    fireEvent.click(screen.getByRole("button", { name: "Scan again" }));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/scans/s2"));
    expect(api.createScan).toHaveBeenCalledWith("example.com", "light", {
      fresh: true,
      verificationToken: undefined,
    });
  });
});

describe("ResultsView for a running scan", () => {
  it("shows what was found so far, without what only a finished scan has", async () => {
    const running = scan("running");
    render(<ResultsView scan={running} result={liveSummary(running)} live />);
    await screen.findByText("example.com", { selector: "span" });

    expect(screen.getByRole("heading", { name: "Found so far" })).toBeTruthy();
    expect(tab(/^Technologies/)).toBeNull();
    // Not in the overview table either.
    expect(screen.queryByRole("button", { name: "Technologies" })).toBeNull();
    expect(screen.queryByRole("link", { name: "Download URLs (CSV)" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Scan again" })).toBeNull();
  });

  it("reloads the lists on request", async () => {
    const running = scan("running");
    render(<ResultsView scan={running} result={liveSummary(running)} live />);
    await screen.findByText("example.com", { selector: "span" });
    expect(api.getHosts).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "Refresh lists" }));
    await waitFor(() => expect(api.getHosts).toHaveBeenCalledTimes(2));
  });

  it("keeps the open tab and reloads its list when the scan finishes", async () => {
    api.getUrls.mockReset().mockResolvedValue({ urls: [] });
    const running = scan("running");
    const { rerender } = render(<ResultsView scan={running} result={liveSummary(running)} live />);
    fireEvent.click(tab(/^APIs/)!);
    await waitFor(() => expect(api.getUrls).toHaveBeenCalledTimes(1));

    rerender(<ResultsView scan={scan("completed")} result={finished} />);
    await waitFor(() => expect(api.getUrls).toHaveBeenCalledTimes(2));
    expect(tab(/^APIs/)?.getAttribute("aria-current")).toBe("page");
  });
});

describe("Hosting tab", () => {
  const withProviders: ScanSummary = {
    ...finished,
    providers: [
      { asn: 13335, name: "Cloudflare, Inc.", country: "US", hosts: 3 },
      { asn: 47583, name: "AS-HOSTINGER", country: "IN", hosts: 1 },
    ],
  };

  it("lists the networks and, when one is opened, its hosts", async () => {
    render(<ResultsView scan={scan("completed")} result={withProviders} />);
    expect(tab(/^Hosting/)?.textContent).toBe("Hosting2");
    fireEvent.click(tab(/^Hosting/)!);

    const hostinger = await screen.findByRole("button", { name: /AS-HOSTINGER \(AS47583\)/ });
    expect(screen.getByRole("button", { name: /Cloudflare, Inc\. \(AS13335\)/ }).textContent).toContain("3 hosts");

    api.getHosts.mockClear();
    fireEvent.click(hostinger);
    // The hosts of that network are asked for by its number.
    await waitFor(() => expect(api.getHosts).toHaveBeenCalled());
    expect(api.getHosts.mock.calls[0][1]).toMatchObject({ asn: 47583 });
  });

  it("explains itself for results without hosting information", () => {
    render(<ResultsView scan={scan("completed")} result={finished} />);
    fireEvent.click(tab(/^Hosting/)!);
    expect(screen.getByText(/No hosting information/)).toBeTruthy();
  });

  it("is not shown while the scan is running", () => {
    render(<ResultsView scan={scan("running")} result={liveSummary(scan("running"))} live />);
    expect(tab(/^Hosting/)).toBeNull();
  });
});

describe("DNS tab, alias note and sharing", () => {
  const full: ScanSummary = {
    ...finished,
    dns: {
      mx: [{ host: "aspmx.l.google.com", priority: 1 }],
      ns: ["bob.ns.cloudflare.com"],
      txt: ["v=spf1 include:_spf.google.com ~all"],
      services: [{ name: "Google Workspace (mail)", evidence: "MX aspmx.l.google.com" }],
    },
    danglingAliases: ["shop.example.com"],
  };

  it("shows the domain's records and the services they reveal", () => {
    render(<ResultsView scan={scan("completed")} result={full} />);
    fireEvent.click(tab(/^DNS/)!);
    expect(screen.getByText("Google Workspace (mail)")).toBeTruthy();
    expect(screen.getByText(/^1\s+aspmx\.l\.google\.com$/)).toBeTruthy();
    expect(screen.getByText("bob.ns.cloudflare.com")).toBeTruthy();
    expect(screen.getByText("v=spf1 include:_spf.google.com ~all")).toBeTruthy();
  });

  it("says so when no records were read", () => {
    render(<ResultsView scan={scan("completed")} result={finished} />);
    fireEvent.click(tab(/^DNS/)!);
    expect(screen.getByText(/No DNS records were read for example.com/)).toBeTruthy();
  });

  it("lists aliases to missing names in the overview and opens the host", async () => {
    render(<ResultsView scan={scan("completed")} result={full} />);
    expect(screen.getByRole("heading", { name: "Aliases to names that do not exist" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "shop.example.com" }));
    // It lands on the Hosts tab, searching for that host.
    await waitFor(() => expect(tab(/^Hosts/)?.getAttribute("aria-current")).toBe("page"));
    expect((screen.getByRole("searchbox", { name: "Search hosts" }) as HTMLInputElement).value).toBe(
      "shop.example.com",
    );
  });

  it("copies the result's link", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    render(<ResultsView scan={scan("completed")} result={finished} />);
    fireEvent.click(screen.getByRole("button", { name: "Copy link" }));
    await screen.findByRole("button", { name: "Link copied" });
    expect(writeText).toHaveBeenCalledWith(window.location.href);
  });
});
