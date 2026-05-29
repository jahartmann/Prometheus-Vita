import { describe, it, expect } from "vitest";
import { toArray, getApiErrorMessage } from "./response";

describe("toArray", () => {
  it("returns a raw array unchanged", () => {
    expect(toArray([1, 2, 3])).toEqual([1, 2, 3]);
  });

  it("unwraps the {success,data:[...]} envelope", () => {
    expect(toArray({ success: true, data: [1, 2] })).toEqual([1, 2]);
  });

  it("returns [] for an empty envelope (backend omitempty dropped the data key)", () => {
    expect(toArray({ success: true })).toEqual([]);
  });

  it("returns [] for null/undefined and non-array payloads", () => {
    expect(toArray(null)).toEqual([]);
    expect(toArray(undefined)).toEqual([]);
    expect(toArray({ data: "not-an-array" })).toEqual([]);
  });
});

describe("getApiErrorMessage", () => {
  it("prefers response.data.message", () => {
    expect(getApiErrorMessage({ response: { data: { message: "boom" } } }, "fb")).toBe("boom");
  });

  it("falls back to response.data.error", () => {
    expect(getApiErrorMessage({ response: { data: { error: "nope" } } }, "fb")).toBe("nope");
  });

  it("uses the fallback when the payload has neither", () => {
    expect(getApiErrorMessage({ response: { data: {} } }, "fb")).toBe("fb");
  });

  it("maps a bare 500 'Internal Server Error' to a helpful backend hint", () => {
    const msg = getApiErrorMessage({ response: { status: 500, data: "Internal Server Error" } }, "fb");
    expect(msg).toContain("Backend");
  });

  it("uses error.message for plain Error instances", () => {
    expect(getApiErrorMessage(new Error("kaputt"), "fb")).toBe("kaputt");
  });
});
