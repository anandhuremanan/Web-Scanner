import Link from "next/link";

interface Props {
  id: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

/**
 * The confirmation every new scan needs: the visitor may scan the domain and
 * accepts the Terms of Use. The server refuses a scan without it, and logs
 * that it was given.
 */
export function TermsConsent({ id, checked, onChange }: Props) {
  return (
    <label htmlFor={id} className="flex cursor-pointer items-start gap-2 text-sm">
      <input
        id={id}
        type="checkbox"
        required
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-1"
      />
      <span>
        I confirm that I own this domain or have explicit permission from its owner to inspect it. I
        agree to the{" "}
        {/* A new tab, so the form keeps what was typed. */}
        <Link
          href="/terms"
          target="_blank"
          rel="noopener"
          className="underline underline-offset-2 hover:text-fg"
        >
          Terms of Use
        </Link>
        .
      </span>
    </label>
  );
}
