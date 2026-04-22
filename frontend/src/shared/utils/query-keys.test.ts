import { describe, expect, it } from "vitest";
import { queryKeys } from "./query-keys";

describe("queryKeys", () => {
  it("builds the download detail key", () => {
    expect(queryKeys.downloadDetail(42)).toEqual(["downloads", "detail", 42]);
  });

  it("builds the storage profile key", () => {
    expect(queryKeys.storageProfile(5)).toEqual(["storageProfile", 5]);
  });
});
