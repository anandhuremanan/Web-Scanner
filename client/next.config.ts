import type { NextConfig } from "next";

// /api/* requests are forwarded to the Go server by src/proxy.ts.

// Security headers for every page.
//
// The Content-Security-Policy that is enforced only covers what cannot break
// the site: no framing (so another site cannot overlay the "Scan again"
// button), no plugins, no changing the base URL or posting forms elsewhere.
// A full policy, listing exactly what scripts and connections the site uses
// (Turnstile, PostHog), is sent as Report-Only: browsers report what it
// would block without blocking anything, so it can be checked in the
// console and enforced once nothing legitimate shows up.
const enforced = [
  "frame-ancestors 'none'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
].join("; ");

const fullPolicy = [
  "default-src 'self'",
  // Next.js inlines its start-up scripts; Turnstile and PostHog load scripts.
  "script-src 'self' 'unsafe-inline' https://challenges.cloudflare.com https://eu-assets.i.posthog.com https://us-assets.i.posthog.com",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob:",
  "font-src 'self'",
  "connect-src 'self' https://eu.i.posthog.com https://us.i.posthog.com https://eu-assets.i.posthog.com https://us-assets.i.posthog.com",
  "frame-src https://challenges.cloudflare.com",
  "worker-src 'self' blob:",
  enforced,
].join("; ");

const securityHeaders = [
  { key: "Content-Security-Policy", value: enforced },
  { key: "Content-Security-Policy-Report-Only", value: fullPolicy },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), payment=(), usb=()" },
];

const nextConfig: NextConfig = {
  poweredByHeader: false,
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
};

export default nextConfig;
