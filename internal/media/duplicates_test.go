package media

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTitleSimilarity(t *testing.T) {
	tests := []struct {
		a, b    string
		similar bool
	}{
		{"Bohemian Rhapsody", "Bohemian Rhapsody", true},
		{"Bohemian Rhapsody", "bohemian rhapsody", true},
		{"Don't Stop Me Now", "Dont Stop Me Now", true},
		{"Don’t Stop Me Now", "Don't Stop Me Now", true},
		{"Bohemian Rhapsody", "Bohemian Rhapsody (Remastered 2011)", true},
		{"Song", "Song (Live at Wembley)", true},
		{"Song", "Song [Acoustic]", true},
		{"Song", "Song - Acoustic", true},
		{"Under Pressure", "Under Pressure (feat. David Bowie)", true},
		{"Under Pressure", "Under Pressure ft. David Bowie", true},
		{"AC/DC Song", "AC⧸DC Song", true},
		{"What Now?", "What Now？", true},
		{"Stairway to Heaven", "Stairway to Heaven Live", true},

		{"Part 1", "Part 2", false},
		{"Love Me Do", "Love Me Tender", false},
		{"Yesterday", "Yesterday Once More", false},
		{"Highway to Hell", "Stairway to Heaven", false},
		{"I Want to Break Free", "I Want It All", false},
		{"(Intro)", "(Outro)", false},
		{"", "Song", false},
	}
	for _, tt := range tests {
		t.Run(tt.a+" | "+tt.b, func(t *testing.T) {
			score := titleSimilarity(tt.a, tt.b)
			assert.Equal(t, tt.similar, score >= similarThreshold, "score %.2f", score)
			assert.InDelta(t, score, titleSimilarity(tt.b, tt.a), 1e-9, "not symmetric")
		})
	}
}

func TestCheckLibrary(t *testing.T) {
	lib := fstest.MapFS{
		"Queen/A Night at the Opera/Bohemian Rhapsody.mp3":                  {},
		"Queen/A Night at the Opera/Love of My Life.opus":                   {},
		"Queen/A Night at the Opera/cover.jpg":                              {},
		"Queen/Greatest Hits/Bohemian Rhapsody (Remastered 2011).m4a":       {},
		"Queen/Live Killers/Love of My Life (Live).mp3":                     {},
		"Queen/Under Pressure.mp3":                                          {},
		"Queen/.hidden/Bohemian Rhapsody.mp3":                               {},
		"David Bowie/Bohemian Rhapsody.mp3":                                 {},
		"Bohemian Rhapsody.mp4":                                             {},
		"Queen/A Night at the Opera/Bohemian Rhapsody (Live).mp3/notes.txt": {},
	}
	queen := func(title, album string) Tags {
		return Tags{Artist: "Queen", Title: title, Album: album}
	}

	tests := []struct {
		name     string
		tags     Tags
		typ      MediaType
		existing string
		similar  []string
	}{
		{
			name:     "same file",
			tags:     queen("Bohemian Rhapsody", "A Night at the Opera"),
			typ:      MediaAudio,
			existing: "Queen/A Night at the Opera/Bohemian Rhapsody.mp3",
			similar:  []string{"Queen/Greatest Hits/Bohemian Rhapsody (Remastered 2011).m4a"},
		},
		{
			name:     "different case in every part",
			tags:     Tags{Artist: "QUEEN", Title: "bohemian rhapsody", Album: "a night at the opera"},
			typ:      MediaAudio,
			existing: "Queen/A Night at the Opera/Bohemian Rhapsody.mp3",
			similar:  []string{"Queen/Greatest Hits/Bohemian Rhapsody (Remastered 2011).m4a"},
		},
		{
			name:     "another audio format of the same song",
			tags:     queen("Love of My Life", "A Night at the Opera"),
			typ:      MediaAudio,
			existing: "Queen/A Night at the Opera/Love of My Life.opus",
			similar:  []string{"Queen/Live Killers/Love of My Life (Live).mp3"},
		},
		{
			name:    "another album: only similar",
			tags:    queen("Bohemian Rhapsody", "Bohemian Rhapsody (Single)"),
			typ:     MediaAudio,
			similar: []string{"Queen/A Night at the Opera/Bohemian Rhapsody.mp3", "Queen/Greatest Hits/Bohemian Rhapsody (Remastered 2011).m4a"},
		},
		{
			name:     "no album",
			tags:     queen("Under Pressure", ""),
			typ:      MediaAudio,
			existing: "Queen/Under Pressure.mp3",
		},
		{
			name: "new song",
			tags: queen("Somebody to Love", "A Day at the Races"),
			typ:  MediaAudio,
		},
		{
			name: "new artist",
			tags: Tags{Artist: "Muse", Title: "Bohemian Rhapsody"},
			typ:  MediaAudio,
		},
		{
			name:     "video",
			tags:     Tags{Title: "bohemian rhapsody"},
			typ:      MediaVideo,
			existing: "Bohemian Rhapsody.mp4",
		},
		{
			name: "a video doesn't match audio",
			tags: Tags{Title: "Under Pressure"},
			typ:  MediaVideo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := CheckLibrary(lib, tt.tags, tt.typ)
			assert.Equal(t, tt.existing, m.Existing)
			if len(tt.similar) == 0 {
				assert.Empty(t, m.Similar)
			} else {
				assert.Equal(t, tt.similar, m.Similar)
			}
		})
	}

	t.Run("at most maxSimilar, most similar first", func(t *testing.T) {
		lib := fstest.MapFS{
			"A/Song (Live).mp3":     {},
			"A/Song (Demo).mp3":     {},
			"A/Song (Acoustic).mp3": {},
			"A/Song (Remix).mp3":    {},
			"A/Song Live.mp3":       {},
		}
		m := CheckLibrary(lib, Tags{Artist: "A", Title: "Song"}, MediaAudio)
		assert.Equal(t, []string{"A/Song (Acoustic).mp3", "A/Song (Demo).mp3", "A/Song (Live).mp3"}, m.Similar)
	})
}

func TestLibraryService(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "Queen"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Queen", "Song.mp3"), nil, 0o600))
	svc := NewLibraryService(fixedDir{dir: dir})

	t.Run("checks each item in order", func(t *testing.T) {
		matches, err := svc.Check(MediaAudio, []Tags{
			{Artist: "Queen", Title: "Other"},
			{Artist: "queen", Title: "song"},
		})
		require.NoError(t, err)
		require.Len(t, matches, 2)
		assert.Empty(t, matches[0].Existing)
		assert.Equal(t, "Queen/Song.mp3", matches[1].Existing)
	})

	t.Run("rejects bad input", func(t *testing.T) {
		_, err := svc.Check("podcast", nil)
		assert.ErrorIs(t, err, ErrInvalidInput)
		_, err = svc.Check(MediaAudio, make([]Tags, maxLibraryChecks+1))
		assert.ErrorIs(t, err, ErrInvalidInput)
		_, err = svc.Check(MediaAudio, []Tags{{Title: "a\x00b"}})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("an unusable folder", func(t *testing.T) {
		_, err := NewLibraryService(fixedDir{err: os.ErrPermission}).Check(MediaAudio, nil)
		assert.ErrorIs(t, err, ErrDownloadDirUnavailable)
	})
}

func TestDownloadService_Duplicates(t *testing.T) {
	audio := func(title string) Request {
		r := validRequest
		r.Tags = Tags{Artist: "Queen", Title: title, Album: "Album"}
		return r
	}
	newLibrary := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "Queen", "Album"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Queen", "Album", "Song.mp3"), nil, 0o600))
		return dir
	}

	t.Run("a file already in the library is skipped without fetching", func(t *testing.T) {
		f := &fakeFetcher{}
		svc := NewDownloadService(f, fixedDir{dir: newLibrary(t)}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		d, err := svc.Start(audio("SONG"))
		require.NoError(t, err)
		assert.Equal(t, []Status{StatusQueued, StatusDownloading, StatusSkipped}, statusesUntilTerminal(t, updates, d.ID))
		assert.Empty(t, f.jobs)
	})

	t.Run("the skip reason names the file", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, fixedDir{dir: newLibrary(t)}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		d, _ := svc.Start(audio("Song"))
		var last Download
		for last = next(t, updates, d.ID); !last.Status.Terminal(); last = next(t, updates, d.ID) {
		}
		assert.Equal(t, StatusSkipped, last.Status)
		assert.Equal(t, "Already in your library: Queen/Album/Song.mp3", last.Error)
	})

	t.Run("similar songs are reported and still downloaded", func(t *testing.T) {
		f := &fakeFetcher{}
		svc := NewDownloadService(f, fixedDir{dir: newLibrary(t)}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		d, _ := svc.Start(audio("Song (Live)"))
		var last Download
		for last = next(t, updates, d.ID); !last.Status.Terminal(); last = next(t, updates, d.ID) {
		}
		assert.Equal(t, StatusFinished, last.Status)
		assert.Equal(t, []string{"Queen/Album/Song.mp3"}, last.Similar)
		assert.Len(t, f.jobs, 1)
	})

	t.Run("two running downloads of the same file: the second is skipped", func(t *testing.T) {
		release, fetching := make(chan struct{}), make(chan struct{}, 1)
		f := &fakeFetcher{fetch: func(ctx context.Context, job FetchJob, onProgress func(Progress)) error {
			fetching <- struct{}{}
			<-release
			return nil
		}}
		svc := NewDownloadService(f, fixedDir{dir: t.TempDir()}, 2)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		first, _ := svc.Start(audio("New Song"))
		// Only a download that got as far as fetching holds its file.
		<-fetching
		second, _ := svc.Start(audio("new song"))
		assert.Equal(t, []Status{StatusQueued, StatusDownloading, StatusSkipped}, statusesUntilTerminal(t, updates, second.ID))

		close(release)
		statuses := statusesUntilTerminal(t, updates, first.ID)
		assert.Equal(t, StatusFinished, statuses[len(statuses)-1])
	})

	t.Run("a released claim can be taken again", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, testDest, 1)
		release, ok := svc.claim("/library", []string{"A", "Song"})
		require.True(t, ok)
		_, ok = svc.claim("/library", []string{"a", "SONG"})
		assert.False(t, ok)
		release()
		_, ok = svc.claim("/library", []string{"A", "Song"})
		assert.True(t, ok)
	})
}
