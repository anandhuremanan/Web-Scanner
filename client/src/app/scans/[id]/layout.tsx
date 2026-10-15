import type { Metadata } from "next";

// Scan results are short-lived and one per scan: keep them out of search.
export const metadata: Metadata = {
  title: "Scan result",
  robots: { index: false, follow: false },
};

export default function ScanLayout({ children }: { children: React.ReactNode }) {
  return children;
}
