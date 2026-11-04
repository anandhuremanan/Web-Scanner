// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import Bot from "./page";

afterEach(cleanup);

describe("the page the scanner's user agent links to", () => {
  it("shows the user agent site owners find in their logs", () => {
    render(<Bot />);
    // The server's default (config.DefaultUserAgent) must stay the same.
    expect(screen.getByText("WebScanner/1.0 (+https://webscanner.imanandhu.in/bot)")).toBeTruthy();
  });

  it("tells them how to have their domain excluded", () => {
    render(<Bot />);
    const contact = screen.getByRole("link", { name: "contact page" });
    expect(contact.getAttribute("href")).toBe("https://www.pflabs.in/contact");
  });
});
