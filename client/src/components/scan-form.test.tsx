// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ createScan: vi.fn() }));
vi.mock("@/lib/api", () => api);
const push = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));
vi.mock("@/lib/verification", () => ({
  useVerification: () => ({ attach: () => {}, getToken: async () => "token" }),
}));
vi.mock("@/lib/analytics", () => ({ track: vi.fn() }));

import { ScanForm } from "./scan-form";

afterEach(cleanup);
beforeEach(() => {
  api.createScan.mockReset().mockResolvedValue({ id: "s1" });
  push.mockReset();
});

function fill(target: string) {
  render(<ScanForm />);
  fireEvent.change(screen.getByLabelText("Domain or URL"), { target: { value: target } });
}

describe("ScanForm", () => {
  it("starts nothing until the visitor confirms they may scan the domain", () => {
    fill("example.com");
    const box = screen.getByRole("checkbox", { name: /I confirm that I own this domain/ });
    expect(box).toHaveProperty("checked", false);
    expect(box).toHaveProperty("required", true);
    // Submitting the form directly skips the browser's own required check.
    fireEvent.submit(screen.getByRole("button", { name: "Map website" }));
    expect(api.createScan).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toMatch(/Tick the box/);
  });

  it("sends the confirmation with the scan", async () => {
    fill("example.com");
    fireEvent.click(screen.getByRole("checkbox", { name: /I confirm that I own this domain/ }));
    fireEvent.submit(screen.getByRole("button", { name: "Map website" }));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/scans/s1"));
    expect(api.createScan).toHaveBeenCalledWith("example.com", "light", {
      verificationToken: "token",
      acceptedTerms: true,
    });
  });

  it("links the Terms of Use in a new tab", () => {
    fill("");
    const link = screen.getByRole("link", { name: "Terms of Use" });
    expect(link.getAttribute("href")).toBe("/terms");
    expect(link.getAttribute("target")).toBe("_blank");
  });
});
