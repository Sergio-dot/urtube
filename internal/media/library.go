package media

import (
	"io/fs"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// maxNameBytes bounds a file or folder name, leaving room for an extension
// within the usual 255-byte limit.
const maxNameBytes = 180

// Fallback names for path components with nothing usable left after
// sanitizing.
const (
	untitled     = "Untitled"
	unknownAlbum = "Unknown Album"
)

// PlanPath returns where a download is saved, as path components relative
// to the library and without the file extension:
//
//	audio: Artist/Album/Title, or Artist/Title when the album is unknown
//	video: Title
//
// Artist and album folders that already exist in lib under a different
// case or Unicode spelling are reused, so "QUEEN" lands in "Queen/".
func PlanPath(lib fs.FS, tags Tags, mediaType MediaType) []string {
	title := SafeName(tags.Title, untitled)
	if mediaType != MediaAudio {
		return []string{title}
	}

	artist := matchExisting(lib, ".", SafeName(tags.Artist, UnknownArtist))
	if strings.TrimSpace(tags.Album) == "" {
		return []string{artist, title}
	}
	album := matchExisting(lib, artist, SafeName(tags.Album, unknownAlbum))
	return []string{artist, album, title}
}

// MatchLibrary returns tags with the artist and album spelled like the
// existing library folders they will be saved in, when the two differ only
// by case, Unicode normalization or spacing. Tags and folders then agree, so
// a library such as Jellyfin shows one "Queen" rather than "Queen" and
// "queen". Names whose folder needed look-alike characters (AC/DC as
// "AC⧸DC") keep their real spelling.
func MatchLibrary(lib fs.FS, tags Tags) Tags {
	artistDir := matchExisting(lib, ".", SafeName(tags.Artist, UnknownArtist))
	if sameName(artistDir, tags.Artist) {
		tags.Artist = artistDir
	}
	if strings.TrimSpace(tags.Album) != "" {
		albumDir := matchExisting(lib, artistDir, SafeName(tags.Album, unknownAlbum))
		if sameName(albumDir, tags.Album) {
			tags.Album = albumDir
		}
	}
	return tags
}

func sameName(a, b string) bool {
	return strings.EqualFold(folderKey(a), folderKey(b))
}

// KnownArtists lists the artist folders in lib, sorted, for suggestions.
func KnownArtists(lib fs.FS, limit int) []string {
	entries, err := fs.ReadDir(lib, ".")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	slices.SortFunc(names, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	if len(names) > limit {
		names = names[:limit]
	}
	return names
}

// matchExisting returns the name of a folder in dir that differs from name
// only by case, Unicode normalization or spacing, or name itself.
func matchExisting(lib fs.FS, dir, name string) string {
	entries, err := fs.ReadDir(lib, dir)
	if err != nil {
		return name
	}
	key := folderKey(name)
	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(folderKey(e.Name()), key) {
			return e.Name()
		}
	}
	return name
}

func folderKey(name string) string {
	return strings.Join(strings.Fields(norm.NFC.String(name)), " ")
}

// windowsUnsafe maps characters that can't appear in file names on Windows
// or SMB shares to look-alikes, as yt-dlp's --windows-filenames does.
var windowsUnsafe = strings.NewReplacer(
	"/", "⧸", `\`, "⧹", ":", "：", "*", "＊", "?", "？",
	`"`, "＂", "<", "＜", ">", "＞", "|", "｜",
)

// windowsReserved are device names Windows refuses as file names.
var windowsReserved = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

// SafeName turns s into a single path component that is valid on Linux,
// Windows and SMB shares, or returns fallback if nothing usable is left.
// It never returns "", "." or "..", and never contains a path separator.
func SafeName(s, fallback string) string {
	s = norm.NFC.String(s)
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' ' // tabs and newlines separate words
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, s)
	s = windowsUnsafe.Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	s = truncateBytes(s, maxNameBytes)
	// Leading dots hide files; trailing dots and spaces are dropped by Windows.
	s = strings.TrimLeft(s, ". ")
	s = strings.TrimRight(s, ". ")

	if s == "" {
		return fallback
	}
	if slices.Contains(windowsReserved, strings.ToUpper(strings.SplitN(s, ".", 2)[0])) {
		s += "_"
	}
	return s
}

// truncateBytes shortens s to at most n bytes without splitting a character.
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !isRuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
