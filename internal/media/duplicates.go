package media

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// LibraryMatch is what the library already holds that a download may
// duplicate. Paths are relative to the library, with their extension.
type LibraryMatch struct {
	// Existing is the file a download would be saved over: the same folder
	// and title, ignoring case, in any format of the same media type. Empty
	// if there is none.
	Existing string
	// Similar lists the artist's other songs whose titles look like the
	// same song, most similar first. Only audio downloads have any.
	Similar []string
}

// maxSimilar bounds LibraryMatch.Similar: it is a hint for the user, not a
// search.
const maxSimilar = 3

// similarThreshold is the token Jaccard similarity from which two titles are
// reported as possibly the same song. Lower values flag unrelated songs that
// share common words ("Love Me Do" and "Love Me Tender").
const similarThreshold = 0.6

// CheckLibrary reports what lib already holds that a download with tags
// would duplicate.
func CheckLibrary(lib fs.FS, tags Tags, mediaType MediaType) LibraryMatch {
	if mediaType == MediaAudio {
		tags = MatchLibrary(lib, tags)
	}
	return checkPath(lib, PlanPath(lib, tags, mediaType), mediaType)
}

// checkPath reports what lib holds that a download saved at planned (see
// PlanPath) would duplicate.
func checkPath(lib fs.FS, planned []string, mediaType MediaType) LibraryMatch {
	m := LibraryMatch{Existing: findExisting(lib, planned, mediaType)}
	// Audio is saved as Artist/[Album/]Title, so the artist's folder holds
	// every version of the song. Videos all share the library's root, where
	// any two titles would be compared.
	if mediaType == MediaAudio && len(planned) > 1 {
		m.Similar = similarTracks(lib, planned)
	}
	return m
}

// findExisting returns the file in lib at planned, in any of mediaType's
// formats, matching the title without regard to case. yt-dlp would otherwise
// treat a same-format file as already downloaded and rewrite its tags and
// cover in place, and a case or format variant would be a second copy.
func findExisting(lib fs.FS, planned []string, mediaType MediaType) string {
	dir := pathpkg.Join(append([]string{"."}, planned[:len(planned)-1]...)...)
	want := planned[len(planned)-1]
	entries, err := fs.ReadDir(lib, dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if stem, ok := mediaFile(e, mediaType); ok && sameName(stem, want) {
			return pathpkg.Join(dir, e.Name())
		}
	}
	return ""
}

// mediaFile returns the name of e without its extension, if e is a file in
// one of mediaType's formats.
func mediaFile(e fs.DirEntry, mediaType MediaType) (string, bool) {
	if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
		return "", false
	}
	ext := pathpkg.Ext(e.Name())
	if !slices.Contains(allowedFormats[mediaType], strings.ToLower(strings.TrimPrefix(ext, "."))) {
		return "", false
	}
	return strings.TrimSuffix(e.Name(), ext), true
}

// similarTracks returns the audio files in the artist folder of planned
// (Artist/[Album/]Title) and in its album folders whose titles look like
// planned's title. The planned file itself, in any format, is left to
// findExisting.
func similarTracks(lib fs.FS, planned []string) []string {
	artist, title := planned[0], planned[len(planned)-1]
	plannedDir := pathpkg.Join(planned[:len(planned)-1]...)

	type candidate struct {
		path  string
		score float64
	}
	var found []candidate
	consider := func(dir string, e fs.DirEntry) {
		stem, ok := mediaFile(e, MediaAudio)
		if !ok || (sameName(dir, plannedDir) && sameName(stem, title)) {
			return
		}
		if score := titleSimilarity(title, stem); score >= similarThreshold {
			found = append(found, candidate{pathpkg.Join(dir, e.Name()), score})
		}
	}

	entries, err := fs.ReadDir(lib, artist)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			consider(artist, e)
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		album := pathpkg.Join(artist, e.Name())
		files, err := fs.ReadDir(lib, album)
		if err != nil {
			continue
		}
		for _, f := range files {
			consider(album, f)
		}
	}

	slices.SortFunc(found, func(a, b candidate) int {
		return cmp.Or(cmp.Compare(b.score, a.score), strings.Compare(a.path, b.path))
	})
	paths := make([]string, 0, min(len(found), maxSimilar))
	for _, c := range found[:min(len(found), maxSimilar)] {
		paths = append(paths, c.path)
	}
	return paths
}

var (
	// featuringCredit matches a featuring credit, with its brackets if any:
	// it names who else performs, not which song it is.
	featuringCredit = regexp.MustCompile(`(?i)[(\[]?\s*\b(?:feat\.?|ft\.?|featuring)\s[^)\]]*[)\]]?`)
	// versionSuffix matches what tells versions of a song apart, such as
	// "(Live at Wembley)", "[Remastered 2011]" or " - Acoustic".
	versionSuffix = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]|\s-\s.*$`)
	// apostrophes are dropped rather than split on, so "Don't" and "Dont"
	// are the same word.
	apostrophes = strings.NewReplacer("'", "", "’", "", "ʼ", "", "`", "")
)

// titleSimilarity scores how likely two song titles name the same song, from
// 0 to 1. It is the Jaccard similarity of their words, ignoring case,
// punctuation and featuring credits, except that titles that differ only by
// a version suffix ("Song" and "Song (Live)") always score at least 0.9:
// whether another version counts as a duplicate is for the user to decide,
// so they are shown it.
func titleSimilarity(a, b string) float64 {
	a, b = featuringCredit.ReplaceAllString(a, " "), featuringCredit.ReplaceAllString(b, " ")
	score := jaccard(titleWords(a), titleWords(b))
	if baseA := titleWords(versionSuffix.ReplaceAllString(a, " ")); len(baseA) > 0 &&
		slices.Equal(baseA, titleWords(versionSuffix.ReplaceAllString(b, " "))) {
		score = max(score, 0.9)
	}
	return score
}

// titleWords returns the distinct words of a title, lowercased and sorted.
// Look-alike characters SafeName put in file names ("⧸", "：") are not
// letters, so they separate words just like the characters they replace.
func titleWords(s string) []string {
	s = apostrophes.Replace(strings.ToLower(norm.NFKC.String(s)))
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	slices.Sort(words)
	return slices.Compact(words)
}

// jaccard returns |a∩b| / |a∪b| for two sorted sets.
func jaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	common := 0
	for _, w := range a {
		if _, ok := slices.BinarySearch(b, w); ok {
			common++
		}
	}
	return float64(common) / float64(len(a)+len(b)-common)
}

// maxLibraryChecks bounds one LibraryService.Check call. Each item reads a
// few folders, and a batch larger than this would be unwieldy to review.
const maxLibraryChecks = 200

// LibraryService reports what the library already holds, so clients can
// warn about duplicates before downloads start.
type LibraryService struct {
	destination Destination
}

// NewLibraryService creates a LibraryService for destination's folder.
func NewLibraryService(destination Destination) *LibraryService {
	return &LibraryService{destination: destination}
}

// Check returns, for each of items, what the library holds that downloading
// it as mediaType would duplicate, in the same order. It is advisory: the
// library can change before a download runs, and downloads check again then.
func (s *LibraryService) Check(mediaType MediaType, items []Tags) ([]LibraryMatch, error) {
	if _, ok := allowedFormats[mediaType]; !ok {
		return nil, ErrInvalidMediaType
	}
	if len(items) > maxLibraryChecks {
		return nil, InvalidInput(fmt.Sprintf("at most %d items can be checked at once", maxLibraryChecks))
	}
	for _, t := range items {
		if err := t.Validate(); err != nil {
			return nil, err
		}
	}
	dir, err := s.destination.DownloadDir()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDownloadDirUnavailable, err)
	}

	lib := os.DirFS(dir)
	matches := make([]LibraryMatch, len(items))
	for i, t := range items {
		matches[i] = CheckLibrary(lib, t, mediaType)
	}
	return matches, nil
}
