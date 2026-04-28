import { api, ApiError } from "@/lib/api/client";
import type { paths } from "@/lib/api/schema";

export type Credentials = paths["/auth/login"]["post"]["requestBody"]["content"]["application/json"];
export type AuthResponse = NonNullable<paths["/auth/login"]["post"]["responses"]["200"]["content"]["application/json"]>;
export type User = NonNullable<paths["/auth/me"]["get"]["responses"]["200"]["content"]["application/json"]>;

export async function loginRequest(credentials: Credentials): Promise<AuthResponse> {
  const { data, error, response } = await api.POST("/auth/login", { body: credentials });
  if (error || !data) {
    throw new ApiError(response?.status ?? 0, error, "Login failed");
  }
  return data;
}

export async function refreshRequest(): Promise<AuthResponse> {
  const { data, error, response } = await api.POST("/auth/refresh", {});
  if (error || !data) {
    throw new ApiError(response?.status ?? 0, error, "Session refresh failed");
  }
  return data;
}

export async function logoutRequest(): Promise<void> {
  await api.POST("/auth/logout", {});
}

export async function getMeRequest(): Promise<User> {
  const { data, error, response } = await api.GET("/auth/me");
  if (error || !data) {
    throw new ApiError(response?.status ?? 0, error, "Not authenticated");
  }
  return data;
}
