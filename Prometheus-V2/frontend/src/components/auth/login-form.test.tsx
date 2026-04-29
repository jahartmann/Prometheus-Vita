import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { LoginForm } from "./login-form";

vi.mock("@/lib/auth/client", () => ({
  loginRequest: vi.fn(),
}));

import { loginRequest } from "@/lib/auth/client";
import { useAuthStore } from "@/lib/auth/store";
import { ApiError } from "@/lib/api/client";

describe("LoginForm", () => {
  beforeEach(() => {
    useAuthStore.getState().clearSession();
    vi.clearAllMocks();
  });

  it("calls onSuccess and stores session on success", async () => {
    (loginRequest as ReturnType<typeof vi.fn>).mockResolvedValue({
      access_token: "tok",
      access_expires_at: new Date().toISOString(),
      user: { id: "uid", email: "a@b.c", name: "Alice", role: "admin", enabled: true },
    });
    const onSuccess = vi.fn();
    render(<LoginForm onSuccess={onSuccess} />);

    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "a@b.c" } });
    fireEvent.change(screen.getByLabelText("Passwort"), { target: { value: "secret-1234" } });
    fireEvent.submit(screen.getByRole("button", { name: /anmelden/i }).closest("form")!);

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledTimes(1);
      expect(useAuthStore.getState().accessToken).toBe("tok");
      expect(useAuthStore.getState().user?.email).toBe("a@b.c");
    });
  });

  it("renders 401 error message on bad credentials", async () => {
    (loginRequest as ReturnType<typeof vi.fn>).mockRejectedValue(new ApiError(401, undefined, "no"));
    render(<LoginForm />);

    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "a@b.c" } });
    fireEvent.change(screen.getByLabelText("Passwort"), { target: { value: "secret-1234" } });
    fireEvent.submit(screen.getByRole("button", { name: /anmelden/i }).closest("form")!);

    await waitFor(() => {
      expect(screen.getByText(/Email oder Passwort falsch/i)).toBeInTheDocument();
    });
  });
});
