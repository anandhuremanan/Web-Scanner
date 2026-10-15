import type { MetadataRoute } from "next";
import { siteUrl } from "@/lib/site";

/** The pages worth finding in search; scan results are left out. */
export default function sitemap(): MetadataRoute.Sitemap {
  return [
    { url: `${siteUrl}/`, changeFrequency: "weekly", priority: 1 },
    { url: `${siteUrl}/cli`, changeFrequency: "monthly", priority: 0.8 },
    { url: `${siteUrl}/terms`, changeFrequency: "yearly", priority: 0.2 },
  ];
}
