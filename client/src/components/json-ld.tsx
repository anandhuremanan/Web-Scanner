/**
 * Structured data (schema.org) that tells search engines and AI assistants
 * what a page is about, in a form they read reliably.
 */
export function JsonLd({ data }: { data: Record<string, unknown> }) {
  return (
    <script
      type="application/ld+json"
      // The data is ours, not user input; "<" is escaped so it cannot end the tag.
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data).replace(/</g, "\\u003c") }}
    />
  );
}
