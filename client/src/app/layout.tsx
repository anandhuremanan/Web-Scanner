import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Analytics } from "@/components/analytics";
import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";
import { siteDescription, siteName, siteUrl } from "@/lib/site";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Web Scanner: find every subdomain and URL of a website",
    template: `%s · ${siteName}`,
  },
  description: siteDescription,
  applicationName: siteName,
  keywords: [
    "subdomain finder",
    "find subdomains",
    "subdomain enumeration",
    "website URL finder",
    "list all pages of a website",
    "certificate transparency search",
    "Wayback Machine URLs",
    "attack surface discovery",
    "DNS records lookup",
    "website map",
  ],
  alternates: { canonical: "/" },
  openGraph: {
    type: "website",
    siteName,
    url: "/",
    title: "Web Scanner: find every subdomain and URL of a website",
    description: siteDescription,
  },
  twitter: {
    card: "summary_large_image",
    title: "Web Scanner: find every subdomain and URL of a website",
    description: siteDescription,
  },
  robots: { index: true, follow: true },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col font-sans">
        <SiteHeader />
        <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-10">{children}</main>
        <SiteFooter />
        <Analytics />
      </body>
    </html>
  );
}
