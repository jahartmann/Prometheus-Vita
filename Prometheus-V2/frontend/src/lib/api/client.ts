import createClient from "openapi-fetch";
import type { paths, components } from "./schema";

export const api = createClient<paths>({
  baseUrl: "/api/v1",
});

export type SystemHealth = NonNullable<
  paths["/system/health"]["get"]["responses"]["200"]["content"]["application/json"]
>;

export type ApiErrorEnvelope = components["schemas"]["Error"];

// ApiError carries the backend's structured error envelope (code, message,
// request_id, optional details) and the HTTP status from the underlying
// fetch Response. React Query's retry discriminator and UI components
// inspect these fields instead of opaque Error messages.
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string;
  readonly details?: Record<string, unknown>;

  constructor(status: number, envelope: ApiErrorEnvelope | undefined, fallback: string) {
    super(envelope?.message ?? fallback);
    this.name = "ApiError";
    this.status = status;
    this.code = envelope?.code ?? "UNKNOWN";
    this.requestId = envelope?.request_id ?? "";
    this.details = envelope?.details;
  }
}
