import { describe, expect, it } from "vitest";
import { ApiError, toApiError, unwrap } from "./error";

const response = (status: number) => new Response(null, { status });

describe("ApiError", () => {
  it("reads ekman's envelope and turns the message into a sentence", () => {
    const e = new ApiError(422, {
      code: "invalid",
      message: "some fields are invalid",
      fields: { "registration.capacity": "must be positive" },
      requestId: "req-1",
    });
    expect(e).toBeInstanceOf(Error);
    expect(e.status).toBe(422);
    expect(e.code).toBe("invalid");
    expect(e.message).toBe("Some fields are invalid.");
    expect(e.fields).toEqual({ "registration.capacity": "must be positive" });
    expect(e.requestId).toBe("req-1");
  });

  it("keeps a message that already ends a sentence", () => {
    expect(new ApiError(409, { message: "Unlist the person instead!" }).message).toBe("Unlist the person instead!");
  });

  it.each([
    [401, "Your session has ended. Sign in again."],
    [403, "Your roles do not allow this."],
    [404, "Not found. It may have been deleted."],
    [503, "The server could not finish that (503). Try again in a moment."],
    [418, "Request failed (418)."],
  ])("falls back to a readable message for %i", (status, message) => {
    const e = new ApiError(status, undefined);
    expect(e.message).toBe(message);
    expect(e.code).toBe(`http_${status}`);
    expect(e.fields).toEqual({});
    expect(e.requestId).toBeUndefined();
  });
});

describe("toApiError", () => {
  it("unwraps the {error: ...} envelope", () => {
    const e = toApiError(409, { error: { code: "conflict", message: "event has RSVPs" } });
    expect(e.code).toBe("conflict");
    expect(e.message).toBe("Event has RSVPs.");
  });

  it("ignores bodies that are not the envelope (a proxy's HTML or text)", () => {
    expect(toApiError(502, "<html>Bad gateway</html>").message).toBe(
      "The server could not finish that (502). Try again in a moment.",
    );
    expect(toApiError(500, null).code).toBe("http_500");
  });
});

describe("unwrap", () => {
  it("returns data on success, including undefined for 204", () => {
    expect(unwrap({ data: { id: "1" }, response: response(200) })).toEqual({ id: "1" });
    expect(unwrap({ data: undefined, response: response(204) })).toBeUndefined();
  });

  it("throws ApiError on an error body or a non-2xx response", () => {
    expect(() => unwrap({ error: { error: { message: "nope" } }, response: response(400) })).toThrow("Nope.");
    expect(() => unwrap({ data: undefined, response: response(500) })).toThrow(ApiError);
  });
});
