// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Command } from "./command";

afterEach(cleanup);

describe("Command", () => {
  it("shows the command and copies it", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    render(<Command label="scan">webscanner example.com</Command>);
    expect(screen.getByText("webscanner example.com")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Copy: scan" }));
    await screen.findByRole("button", { name: "Copy: scan" });
    expect(writeText).toHaveBeenCalledWith("webscanner example.com");
    expect(await screen.findByText("Copied")).toBeTruthy();
  });
});
