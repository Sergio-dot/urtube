import { describe, expect, it } from "vitest";
import type { Video } from "./api";
import {
  checkable,
  flagged,
  hasMatch,
  leftOutByDefault,
  libraryPath,
} from "./library";

const video = (id: string, title?: string): Video => ({
  id,
  title: id,
  uploader: "",
  duration: 0,
  thumbnails: [],
  url: `https://www.youtube.com/watch?v=${id}`,
  isLive: false,
  tags: title ? { artist: "Queen", title, album: "" } : undefined,
});

describe("libraryPath", () => {
  it("separates folders like savedAs", () => {
    expect(libraryPath("Queen/A Night at the Opera/Song.mp3")).toBe(
      "Queen / A Night at the Opera / Song.mp3",
    );
  });
});

describe("hasMatch", () => {
  it("is false for nothing found", () => {
    expect(hasMatch(undefined)).toBe(false);
    expect(hasMatch({})).toBe(false);
    expect(hasMatch({ similar: [] })).toBe(false);
  });

  it("is true for an existing file or similar songs", () => {
    expect(hasMatch({ existing: "A/Song.mp3" })).toBe(true);
    expect(hasMatch({ similar: ["A/Song (Live).mp3"] })).toBe(true);
  });
});

describe("checkable", () => {
  it("keeps the videos whose tags are known, in order", () => {
    const got = checkable([video("a", "One"), video("b"), video("c", "Two")]);
    expect(got.map(({ video }) => video.id)).toEqual(["a", "c"]);
    expect(got.map(({ tags }) => tags.title)).toEqual(["One", "Two"]);
  });
});

describe("flagged and leftOutByDefault", () => {
  const checked = [video("a", "One"), video("b", "Two"), video("c", "Three")];
  const items = flagged(checked, [
    { existing: "Queen/One.mp3" },
    {},
    { similar: ["Queen/Three (Live).mp3"] },
  ]);

  it("pairs each video with its match and drops the clean ones", () => {
    expect(items.map(({ video }) => video.id)).toEqual(["a", "c"]);
    expect(items[0].match.existing).toBe("Queen/One.mp3");
  });

  it("leaves out files already in the library, not similar songs", () => {
    expect([...leftOutByDefault(items)]).toEqual(["a"]);
  });

  it("tolerates fewer matches than videos", () => {
    expect(flagged(checked, [])).toEqual([]);
  });
});
