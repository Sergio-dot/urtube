package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuessTags(t *testing.T) {
	tests := []struct {
		name string
		info VideoInfo
		want Tags
	}{
		{
			name: "YouTube Music metadata wins",
			info: VideoInfo{
				Title: "Bohemian Rhapsody", Channel: "Queen Official",
				Track: "Bohemian Rhapsody", Artists: []string{"Queen"}, Album: "A Night At The Opera", ReleaseYear: 1975,
			},
			want: Tags{Artist: "Queen", Title: "Bohemian Rhapsody", Album: "A Night At The Opera", Year: 1975},
		},
		{
			name: "extra YouTube Music artists are featured",
			info: VideoInfo{Track: "Under Pressure", Artists: []string{"Queen", "David Bowie"}},
			want: Tags{Artist: "Queen", Title: "Under Pressure (feat. David Bowie)"},
		},
		{
			name: "songwriter credits are not featured, duplicates dropped",
			info: VideoInfo{Track: "Mashup", Artists: []string{"nimphia", "klyron", "A", "B", "C", "klyron", "A"}},
			want: Tags{Artist: "nimphia", Title: "Mashup"},
		},
		{
			name: "up to three others are featured",
			info: VideoInfo{Track: "Song", Artists: []string{"Alex Goot", "Sam Tsui", "sam tsui", "Tyler Ward", "Kurt"}},
			want: Tags{Artist: "Alex Goot", Title: "Song (feat. Sam Tsui, Tyler Ward, Kurt)"},
		},
		{
			name: "en dash title with video noise",
			info: VideoInfo{Title: "Queen – Bohemian Rhapsody (Official Video Remastered)", Channel: "Queen Official"},
			want: Tags{Artist: "Queen", Title: "Bohemian Rhapsody"},
		},
		{
			name: "several noise groups, recording details kept",
			info: VideoInfo{Title: "Rick Astley - Never Gonna Give You Up (Official Video) (4K Remaster)", Channel: "Rick Astley"},
			want: Tags{Artist: "Rick Astley", Title: "Never Gonna Give You Up"},
		},
		{
			name: "live performance stays in the title",
			info: VideoInfo{Title: "Benson Boone - Bohemian Rhapsody (Live at Coachella 2025)", Channel: "Benson Boone"},
			want: Tags{Artist: "Benson Boone", Title: "Bohemian Rhapsody (Live at Coachella 2025)"},
		},
		{
			name: "remaster note stays in the title",
			info: VideoInfo{Title: "Bohemian Rhapsody (Remastered 2011)", Channel: "Queen - Topic"},
			want: Tags{Artist: "Queen", Title: "Bohemian Rhapsody (Remastered 2011)"},
		},
		{
			name: "featured artist moves to the title",
			info: VideoInfo{Title: "Calvin Harris feat. Rihanna - This Is What You Came For [Official Video]"},
			want: Tags{Artist: "Calvin Harris", Title: "This Is What You Came For (feat. Rihanna)"},
		},
		{
			name: "ft. with an existing credit in the title",
			info: VideoInfo{Title: "Artist ft. Other - Song (feat. Other)"},
			want: Tags{Artist: "Artist", Title: "Song (feat. Other)"},
		},
		{
			name: "band names with & or , are not split",
			info: VideoInfo{Title: "Simon & Garfunkel - The Sound of Silence (Audio)"},
			want: Tags{Artist: "Simon & Garfunkel", Title: "The Sound of Silence"},
		},
		{
			name: "no dash: artist from a VEVO channel",
			info: VideoInfo{Title: "Thunderstruck (Official Video)", Channel: "acdcVEVO"},
			want: Tags{Artist: "acdc", Title: "Thunderstruck"},
		},
		{
			name: "hyphen inside a word is not a separator",
			info: VideoInfo{Title: "Spider-Man Theme", Channel: "Composer"},
			want: Tags{Artist: "Composer", Title: "Spider-Man Theme"},
		},
		{
			name: "falls back to the uploader, then Unknown Artist",
			info: VideoInfo{Title: "Untitled jam", Uploader: "Someone"},
			want: Tags{Artist: "Someone", Title: "Untitled jam"},
		},
		{
			name: "nothing known",
			info: VideoInfo{Title: "Lyrics"},
			want: Tags{Artist: UnknownArtist, Title: "Lyrics"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GuessTags(tt.info))
		})
	}
}

func TestArtistCredit(t *testing.T) {
	assert.Equal(t, "Queen", ArtistCredit([]string{"Queen"}))
	assert.Equal(t, "A, B, C, D", ArtistCredit([]string{"A", "B", "C", "D"}))
	assert.Equal(t, "A, B, C and 3 more", ArtistCredit([]string{"A", "B", "C", "D", "E", "F", "a"}))
	assert.Equal(t, "", ArtistCredit(nil))
}

func TestTags(t *testing.T) {
	t.Run("Or fills only empty fields", func(t *testing.T) {
		got := Tags{Artist: " Queen ", Title: ""}.Or(Tags{Artist: "Guess", Title: "Song", Album: "Album", Year: 1975})
		assert.Equal(t, Tags{Artist: "Queen", Title: "Song", Album: "Album", Year: 1975}, got)
	})

	t.Run("Validate", func(t *testing.T) {
		assert.NoError(t, Tags{Artist: "Queen", Year: 1975}.Validate())
		assert.ErrorIs(t, Tags{Title: strings.Repeat("a", 201)}.Validate(), ErrInvalidInput)
		assert.ErrorIs(t, Tags{Album: "a\nb"}.Validate(), ErrInvalidInput)
		assert.ErrorIs(t, Tags{Year: 75}.Validate(), ErrInvalidInput)
	})
}

func TestPlanPath(t *testing.T) {
	lib := fstest.MapFS{
		"Queen/A Night At The Opera/Bohemian Rhapsody.mp3": {},
		"Simon & Garfunkel/x.mp3":                          {},
		"notes.txt":                                        {},
	}

	tests := []struct {
		name  string
		tags  Tags
		media MediaType
		want  []string
	}{
		{"audio with album", Tags{Artist: "Queen", Album: "News of the World", Title: "We Will Rock You"}, MediaAudio,
			[]string{"Queen", "News of the World", "We Will Rock You"}},
		{"audio without album", Tags{Artist: "Queen", Title: "Don't Stop Me Now"}, MediaAudio,
			[]string{"Queen", "Don't Stop Me Now"}},
		{"existing artist and album reused regardless of case and spacing",
			Tags{Artist: "QUEEN ", Album: "a night  at the opera", Title: "Love of My Life"}, MediaAudio,
			[]string{"Queen", "A Night At The Opera", "Love of My Life"}},
		{"a file is not an artist folder", Tags{Artist: "Notes.txt", Title: "x"}, MediaAudio,
			[]string{"Notes.txt", "x"}},
		{"video is flat", Tags{Artist: "Queen", Album: "x", Title: "Live Aid 1985"}, MediaVideo,
			[]string{"Live Aid 1985"}},
		{"separators can't escape the library", Tags{Artist: "../..", Title: "a/b"}, MediaAudio,
			[]string{"⧸", "a⧸b"}},
		{"AC/DC", Tags{Artist: "AC/DC", Title: "Thunderstruck"}, MediaAudio,
			[]string{"AC⧸DC", "Thunderstruck"}},
		{"empty tags", Tags{}, MediaAudio, []string{UnknownArtist, untitled}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PlanPath(lib, tt.tags, tt.media))
		})
	}

	t.Run("missing library", func(t *testing.T) {
		assert.Equal(t, []string{"Queen", "x"}, PlanPath(fstest.MapFS{}, Tags{Artist: "Queen", Title: "x"}, MediaAudio))
	})
}

func TestMatchLibrary(t *testing.T) {
	lib := fstest.MapFS{
		"QUEEN/A Night At The Opera/x.mp3": {},
		"AC⧸DC/x.mp3":                      {},
	}
	tests := []struct {
		name string
		in   Tags
		want Tags
	}{
		{"adopts folder spelling", Tags{Artist: "queen", Album: "a night at the opera", Title: "x"},
			Tags{Artist: "QUEEN", Album: "A Night At The Opera", Title: "x"}},
		{"unknown names unchanged", Tags{Artist: "ABBA", Album: "Arrival"}, Tags{Artist: "ABBA", Album: "Arrival"}},
		{"look-alike folders keep the real name", Tags{Artist: "ac/dc"}, Tags{Artist: "ac/dc"}},
		{"no album", Tags{Artist: "Queen"}, Tags{Artist: "QUEEN"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchLibrary(lib, tt.in))
		})
	}
}

func TestSafeName(t *testing.T) {
	tests := map[string]string{
		"Bohemian Rhapsody":      "Bohemian Rhapsody",
		`What? "Yes": <no> | *`:  "What？ ＂Yes＂： ＜no＞ ｜ ＊",
		"  spaced\tout  ":        "spaced out",
		".hidden":                "hidden",
		"trailing dots...":       "trailing dots",
		"..":                     "fallback",
		"":                       "fallback",
		"con":                    "con_",
		"Nul.mp3":                "Nul.mp3_",
		"line\nbreak":            "line break",
		"bell\x07":               "bell",
		strings.Repeat("é", 200): strings.Repeat("é", 90),
		"Beyoncé decomposed":    "Beyoncé decomposed",
	}
	for in, want := range tests {
		assert.Equal(t, want, SafeName(in, "fallback"), "%q", in)
	}
}

func TestKnownArtists(t *testing.T) {
	lib := fstest.MapFS{
		"queen/x.mp3":   {},
		"ABBA/x.mp3":    {},
		".hidden/x.mp3": {},
		"file.mp3":      {},
	}
	assert.Equal(t, []string{"ABBA", "queen"}, KnownArtists(lib, 10))
	assert.Equal(t, []string{"ABBA"}, KnownArtists(lib, 1))
	assert.Nil(t, KnownArtists(fstest.MapFS{}, 10))
}

func TestDownloadService_TagsAndPath(t *testing.T) {
	lib := t.TempDir()
	f := &fakeFetcher{info: VideoInfo{Title: "Queen – Bohemian Rhapsody (Official Video)", Channel: "Queen Official"}}
	svc := NewDownloadService(f, fixedDir{dir: lib}, 1)
	_, updates, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	req := validRequest
	req.Tags = Tags{Album: "A Night At The Opera"}
	d, err := svc.Start(req)
	require.NoError(t, err)
	statusesUntilTerminal(t, updates, d.ID)

	f.mu.Lock()
	defer f.mu.Unlock()
	require.Len(t, f.jobs, 1)
	job := f.jobs[0]
	assert.Equal(t, Tags{Artist: "Queen", Title: "Bohemian Rhapsody", Album: "A Night At The Opera"}, job.Tags,
		"client tags fill in, the rest is guessed")
	assert.Equal(t, []string{"Queen", "A Night At The Opera", "Bohemian Rhapsody"}, job.Path)
	assert.Equal(t, lib, job.Dir)
	assert.True(t, job.TrimNonMusic, "audio downloads trim non-music segments")
}

func TestDownloadService_InspectError(t *testing.T) {
	svc := NewDownloadService(&fakeFetcher{inspectErr: errors.New("Video unavailable")}, testDest, 1)
	_, updates, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	d, err := svc.Start(validRequest)
	require.NoError(t, err)
	statuses := statusesUntilTerminal(t, updates, d.ID)
	assert.Equal(t, StatusError, statuses[len(statuses)-1])
}

func mkdirs(root string, names ...string) error {
	for _, n := range names {
		if err := os.Mkdir(filepath.Join(root, n), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func TestSuggestService(t *testing.T) {
	lib := t.TempDir()
	require.NoError(t, mkdirs(lib, "Queen", "ABBA"))

	t.Run("guesses tags and lists artists", func(t *testing.T) {
		svc := NewSuggestService(&fakeFetcher{info: VideoInfo{Title: "Queen - Bohemian Rhapsody"}}, fixedDir{dir: lib})
		got, err := svc.Suggest(context.Background(), "https://example.com/v")
		require.NoError(t, err)
		assert.Equal(t, Tags{Artist: "Queen", Title: "Bohemian Rhapsody"}, got.Tags)
		assert.Equal(t, []string{"ABBA", "Queen"}, got.KnownArtists)
	})

	t.Run("an unusable folder only drops the artist list", func(t *testing.T) {
		svc := NewSuggestService(&fakeFetcher{info: VideoInfo{Title: "Song"}}, fixedDir{err: errors.New("not writable")})
		got, err := svc.Suggest(context.Background(), "https://example.com/v")
		require.NoError(t, err)
		assert.Nil(t, got.KnownArtists)
	})

	t.Run("rejects invalid URLs", func(t *testing.T) {
		svc := NewSuggestService(&fakeFetcher{}, testDest)
		_, err := svc.Suggest(context.Background(), "--exec=id")
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("lookup failures are ErrVideoUnavailable", func(t *testing.T) {
		svc := NewSuggestService(&fakeFetcher{inspectErr: errors.New("Private video")}, testDest)
		_, err := svc.Suggest(context.Background(), "https://example.com/v")
		assert.ErrorIs(t, err, ErrVideoUnavailable)
		assert.EqualError(t, err, "could not read the video: Private video")
	})
}

// countingNotifier counts LibraryChanged calls.
type countingNotifier struct{ calls chan struct{} }

func (n countingNotifier) LibraryChanged(ctx context.Context) error {
	n.calls <- struct{}{}
	return nil
}

func TestDownloadService_NotifiesLibraryOnceIdle(t *testing.T) {
	release := make(chan struct{})
	svc := NewDownloadService(blockingFetcher(release), testDest, 1)
	notifier := countingNotifier{calls: make(chan struct{}, 10)}
	svc.SetLibraryNotifier(notifier)
	_, updates, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	// Three downloads, one at a time: one notification at the end.
	var ids []string
	for range 3 {
		d, err := svc.Start(validRequest)
		require.NoError(t, err)
		ids = append(ids, d.ID)
	}
	close(release)
	for _, id := range ids {
		statusesUntilTerminal(t, updates, id)
	}

	select {
	case <-notifier.calls:
	case <-time.After(2 * time.Second):
		t.Fatal("library not notified")
	}
	select {
	case <-notifier.calls:
		t.Fatal("notified more than once for one batch")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestDownloadService_NoNotificationWithoutFinishedDownloads(t *testing.T) {
	svc := NewDownloadService(&fakeFetcher{inspectErr: errors.New("gone")}, testDest, 1)
	notifier := countingNotifier{calls: make(chan struct{}, 10)}
	svc.SetLibraryNotifier(notifier)
	_, updates, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	d, err := svc.Start(validRequest)
	require.NoError(t, err)
	statusesUntilTerminal(t, updates, d.ID)

	select {
	case <-notifier.calls:
		t.Fatal("a failed download must not trigger a library scan")
	case <-time.After(200 * time.Millisecond):
	}
}
