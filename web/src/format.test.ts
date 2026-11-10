import { describe, expect, it } from "vitest";
import { formatDuration, percentValue } from "./format";

describe("formatDuration", () => {
  it.each([
    [0, "0:00"],
    [5, "0:05"],
    [65, "1:05"],
    [3600, "1:00:00"],
    [3725, "1:02:05"],
  ])("%d → %s", (seconds, want) => {
    expect(formatDuration(seconds)).toBe(want);
  });
});

describe("percentValue", () => {
  it.each([
    ["42.5%", 42.5],
    ["  7%", 7],
    ["100.0%", 100],
    ["120%", 100],
    ["-3%", 0],
  ])("%j → %d", (percent, want) => {
    expect(percentValue(percent)).toBe(want);
  });

  it.each(["", "N/A", "%"])("%j has no value", (percent) => {
    expect(percentValue(percent)).toBeUndefined();
  });
});
