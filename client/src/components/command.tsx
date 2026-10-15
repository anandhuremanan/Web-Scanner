"use client";

import { useState } from "react";

/** A command to type, shown in a box with a button that copies it. */
export function Command({ children, label }: { children: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // No clipboard access: the command can still be selected by hand.
    }
  }

  return (
    <div className="flex items-start gap-2 rounded-md border border-border bg-surface px-3 py-2">
      <pre className="min-w-0 flex-1 overflow-x-auto whitespace-pre-wrap break-all font-mono text-xs leading-5">
        {children}
      </pre>
      <button
        type="button"
        onClick={copy}
        aria-label={label ? `Copy: ${label}` : "Copy command"}
        className="shrink-0 rounded px-2 py-0.5 text-xs text-accent hover:bg-subtle"
      >
        {copied ? "Copied" : "Copy"}
      </button>
    </div>
  );
}
