import type { MetadataRoute } from "next";
import { siteUrl } from "@/lib/site";

/**
 * Search engines and AI assistants may read every page except scan results
 * (short-lived, one per scan) and the API.
 */
export default function robots(): MetadataRoute.Robots {
  const disallow = ["/scans/", "/api/"];
  return {
    rules: [
      { userAgent: "*", allow: "/", disallow },
      // Named explicitly so that the site is welcome in AI search answers
      // even where a crawler's defaults are cautious.
      {
        userAgent: ["GPTBot", "OAI-SearchBot", "ChatGPT-User", "ClaudeBot", "Claude-SearchBot", "PerplexityBot", "Google-Extended", "Applebot-Extended"],
        allow: "/",
        disallow,
      },
    ],
    sitemap: `${siteUrl}/sitemap.xml`,
    host: siteUrl,
  };
}
