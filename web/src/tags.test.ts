import { describe, expect, it } from "vitest";
import { savedAs } from "./tags";

describe("savedAs", () => {
  const tags = {
    artist: "Queen",
    title: "Bohemian Rhapsody",
    album: "A Night At The Opera",
  };

  it("audio with album", () => {
    expect(savedAs(tags, true, "mp3")).toBe(
      "Queen / A Night At The Opera / Bohemian Rhapsody.mp3",
    );
  });

  it("audio without album", () => {
    expect(savedAs({ ...tags, album: " " }, true, "m4a")).toBe(
      "Queen / Bohemian Rhapsody.m4a",
    );
  });

  it("empty fields show the fallbacks", () => {
    expect(savedAs({ artist: "", title: "", album: "" }, true, "mp3")).toBe(
      "Unknown Artist / Untitled.mp3",
    );
  });

  it("video is just the title", () => {
    expect(savedAs(tags, false, "mkv")).toBe("Bohemian Rhapsody.mkv");
  });
});
