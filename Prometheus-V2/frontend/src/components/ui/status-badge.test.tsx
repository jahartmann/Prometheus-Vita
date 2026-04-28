import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./status-badge";

describe("StatusBadge", () => {
  it("renders children with the chosen tone", () => {
    render(<StatusBadge tone="ok">Cluster operativ</StatusBadge>);
    const el = screen.getByText("Cluster operativ");
    expect(el).toBeInTheDocument();
    expect(el.getAttribute("data-tone")).toBe("ok");
  });

  it("renders without icon when withIcon=false", () => {
    render(
      <StatusBadge tone="warning" withIcon={false}>
        Achtung
      </StatusBadge>
    );
    const el = screen.getByText("Achtung");
    expect(el.querySelector("svg")).toBeNull();
  });
});
