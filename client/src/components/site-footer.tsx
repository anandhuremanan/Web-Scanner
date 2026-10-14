import Link from "next/link";

export const sourceUrl = "https://github.com/anandhuremanan/website_mapper";
export const authorUrl = "https://imanandhu.in";

/** Credits and links shown at the bottom of every page. */
export function SiteFooter() {
  return (
    <footer className="print:hidden border-t border-border">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-x-6 gap-y-2 px-4 py-5 text-xs text-muted">
        <p>
          Made by{" "}
          <a href={authorUrl} target="_blank" rel="noopener noreferrer" className="text-fg hover:underline">
            Anandhu Remanan
          </a>
        </p>
        <p className="flex flex-wrap gap-x-4 gap-y-1">
          <Link href="/terms" className="hover:text-fg hover:underline">
            Terms &amp; Privacy
          </Link>
          <a href={sourceUrl} target="_blank" rel="noopener noreferrer" className="hover:text-fg hover:underline">
            Source on GitHub
          </a>
          <a
            href={`${sourceUrl}/blob/main/LICENSE`}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-fg hover:underline"
          >
            MIT License
          </a>
        </p>
      </div>
    </footer>
  );
}
