import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { HealthStatus } from "./health-status";

vi.mock("@/lib/api/client", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api/client")>("@/lib/api/client");
  return {
    ...actual,
    api: {
      GET: vi.fn(),
    },
  };
});

import { api } from "@/lib/api/client";

describe("HealthStatus", () => {
  let qc: QueryClient;
  beforeEach(() => {
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    vi.clearAllMocks();
  });

  function renderWith() {
    return render(
      <QueryClientProvider client={qc}>
        <HealthStatus />
      </QueryClientProvider>
    );
  }

  it("renders status and version on success", async () => {
    (api.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: { status: "ok", version: "0.1.0", request_id: "rid" },
      error: undefined,
    });

    renderWith();
    await waitFor(() => {
      expect(screen.getByTestId("health-status").textContent).toContain("status=ok");
      expect(screen.getByTestId("health-status").textContent).toContain("version=0.1.0");
    });
  });

  it("renders error state when backend fails", async () => {
    (api.GET as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: undefined,
      error: { code: "X", message: "down", request_id: "rid-42" },
      response: { status: 503 },
    });

    renderWith();
    await waitFor(() => {
      const text = screen.getByTestId("health-status").textContent ?? "";
      expect(text).toContain("Backend nicht erreichbar");
      expect(text).toContain("rid-42");
    });
  });
});
