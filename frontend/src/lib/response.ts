// Pure API-response helpers, kept free of axios/store imports so they can be
// unit-tested in isolation (and imported without side effects).

/**
 * Safely extract an array from an API response, handling a raw array, the
 * {success,data} envelope, and an already-interceptor-unwrapped payload.
 * Defends against the backend's `omitempty` on empty lists (which drops the
 * `data` key entirely) by falling back to an empty array.
 */
export function toArray<T>(data: unknown): T[] {
  if (Array.isArray(data)) return data;
  if (data && typeof data === "object" && "data" in (data as Record<string, unknown>)) {
    const inner = (data as Record<string, unknown>).data;
    if (Array.isArray(inner)) return inner;
  }
  return [];
}

/** Extract a human-readable message from an API/Axios error, with a fallback. */
export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === "object" && "response" in error) {
    const response = (error as { response?: { status?: number; data?: unknown } }).response;
    const data = response?.data;

    if (data && typeof data === "object") {
      const payload = data as { message?: string; error?: string };
      return payload.message ?? payload.error ?? fallback;
    }

    if (typeof data === "string" && data.trim() && data.trim() !== "Internal Server Error") {
      return data;
    }

    if (response?.status === 500 && data === "Internal Server Error") {
      return "API momentan nicht erreichbar. Bitte pruefen, ob das Backend auf Port 8080 laeuft.";
    }

    return fallback;
  }
  if (error instanceof Error) return error.message;
  return fallback;
}
