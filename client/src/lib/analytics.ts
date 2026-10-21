import posthog, { type CaptureResult } from "posthog-js";

// Usage analytics (PostHog), to see how the tool is used: how many scans are
// started, in which mode, and what people open afterwards.
//
// It is off unless NEXT_PUBLIC_POSTHOG_KEY is set. When on, it is set up to
// collect as little as it can:
//
//   - no cookies and nothing stored in the browser, so no consent banner is
//     needed for it; each page load counts as a new anonymous visitor
//   - no automatic capture of clicks or text, which would pick up the
//     hostnames shown on result pages
//   - no session recording
//
// The scanned domain, hosts and URLs are never sent: which sites people
// look up is exactly what this tool should not report to anyone. Events
// carry only the properties listed in Events below.

/** Every event the client sends, with its properties. */
export interface Events {
  /** A scan was requested from the form. */
  scan_started: {
    mode: string;
    /** "new", "joined" (an equivalent scan was running) or "reused" (a recent result). */
    outcome: "new" | "joined" | "reused";
  };
  /** "Scan again" was used on a result. */
  scan_again: { mode: string };
  /** A scan this tab was watching reached its end. */
  scan_finished: {
    mode: string;
    status: string;
    stop_reason: string;
    duration_s: number;
    hosts: number;
    urls: number;
  };
  /** A running scan was cancelled from the progress panel. */
  scan_cancelled: { mode: string };
  /** A result tab was opened. */
  result_tab_opened: { tab: string; live: boolean };
  /** The CSV of a result was downloaded. */
  urls_exported: { urls: number };
  /** A result's link was copied, or the result was printed. */
  result_shared: { how: "link" | "print" };
}

let enabled = false;

/** A result page's address with its scan ID. */
const SCAN_PATH = /\/scans\/[^/?#\s]+/g;

/**
 * Removes scan IDs from an event before it is sent. Page views carry the
 * page's address, and a result page's address holds the scan's ID, which
 * leads to the scanned domain for anyone who opens it. Every string
 * property is rewritten to the route itself: /scans/abc123 -> /scans/[id].
 */
export function withoutScanIds(event: CaptureResult | null): CaptureResult | null {
  if (!event?.properties) return event;
  for (const [key, value] of Object.entries(event.properties)) {
    if (typeof value === "string" && value.includes("/scans/")) {
      event.properties[key] = value.replace(SCAN_PATH, "/scans/[id]");
    }
  }
  return event;
}

/** Starts analytics if a project key is configured. Safe to call again. */
export function initAnalytics() {
  const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;
  if (enabled || !key || typeof window === "undefined") return;
  posthog.init(key, {
    // EU by default, as the privacy page says; NEXT_PUBLIC_POSTHOG_HOST
    // overrides it for a project in another region.
    api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://eu.i.posthog.com",
    persistence: "memory",
    autocapture: false,
    capture_pageview: "history_change",
    capture_pageleave: true,
    disable_session_recording: true,
    disable_surveys: true,
    person_profiles: "identified_only",
    before_send: withoutScanIds,
  });
  enabled = true;
}

/** Records an event; does nothing while analytics is off. */
export function track<E extends keyof Events>(event: E, properties: Events[E]) {
  if (!enabled) return;
  try {
    posthog.capture(event, properties);
  } catch {
    // Analytics must never break the page.
  }
}
