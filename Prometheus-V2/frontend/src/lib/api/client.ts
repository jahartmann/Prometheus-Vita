import createClient from "openapi-fetch";
import type { paths } from "./schema";

export const api = createClient<paths>({
  baseUrl: "/api/v1",
});

export type SystemHealth = NonNullable<
  paths["/system/health"]["get"]["responses"]["200"]["content"]["application/json"]
>;
