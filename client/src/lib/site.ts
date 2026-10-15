/**
 * The site's public address, used for canonical links, the sitemap and
 * social previews. Set NEXT_PUBLIC_SITE_URL for another deployment.
 */
export const siteUrl = (process.env.NEXT_PUBLIC_SITE_URL ?? "https://webscanner.imanandhu.in").replace(/\/$/, "");

export const siteName = "Web Scanner";

/** One sentence that says what the site does, for search results. */
export const siteDescription =
  "Find every subdomain and URL of a website from public sources: certificate transparency logs, subdomain databases, the Wayback Machine, DNS and sitemaps. Free, passive, with the source of every result.";
