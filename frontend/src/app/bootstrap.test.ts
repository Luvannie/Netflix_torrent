import { describe, expect, it } from "vitest";
import { decideInitialRoute } from "./bootstrap";

describe("decideInitialRoute", () => {
  it("routes to startup while bootstrap is not ready", () => {
    expect(
      decideInitialRoute({
        step: "WAITING_HEALTH",
        message: "Waiting for backend",
        backendUrl: "http://127.0.0.1:18080",
        webSocketUrl: "ws://127.0.0.1:18080/ws",
      }),
    ).toBe("/startup");
  });

  it("routes to the app shell when backend is ready", () => {
    expect(
      decideInitialRoute({
        step: "READY",
        message: "Ready",
        backendUrl: "http://127.0.0.1:18080",
        webSocketUrl: "ws://127.0.0.1:18080/ws",
      }),
    ).toBe("/app/catalog");
  });
});

