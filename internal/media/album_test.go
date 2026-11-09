package media

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFinder struct {
	album Album
	err   error

	mu    sync.Mutex
	asked []Tags
}

func (f *fakeFinder) FindAlbum(ctx context.Context, tags Tags) (Album, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, tags)
	return f.album, f.err
}

func TestCorrectAlbum(t *testing.T) {
	ctx := context.Background()
	single := Tags{Artist: "System Of A Down", Title: "Chop Suey!", Album: "Chop Suey!", Year: 2000}

	t.Run("replaces the album and year", func(t *testing.T) {
		got := correctAlbum(ctx, &fakeFinder{album: Album{Title: "Toxicity", Year: 2001}}, single)
		assert.Equal(t, Tags{Artist: "System Of A Down", Title: "Chop Suey!", Album: "Toxicity", Year: 2001}, got)
	})

	t.Run("an unknown year keeps the existing one", func(t *testing.T) {
		got := correctAlbum(ctx, &fakeFinder{album: Album{Title: "Toxicity"}}, single)
		assert.Equal(t, "Toxicity", got.Album)
		assert.Equal(t, 2000, got.Year)
	})

	t.Run("no answer changes nothing", func(t *testing.T) {
		assert.Equal(t, single, correctAlbum(ctx, &fakeFinder{}, single))
	})

	t.Run("a failure changes nothing", func(t *testing.T) {
		assert.Equal(t, single, correctAlbum(ctx, &fakeFinder{err: errors.New("down")}, single))
	})

	t.Run("no finder changes nothing", func(t *testing.T) {
		assert.Equal(t, single, correctAlbum(ctx, nil, single))
	})

	t.Run("an unknown artist isn't looked up", func(t *testing.T) {
		f := &fakeFinder{album: Album{Title: "Wrong"}}
		tags := Tags{Artist: UnknownArtist, Title: "Song"}
		assert.Equal(t, tags, correctAlbum(ctx, f, tags))
		assert.Empty(t, f.asked)
	})

	t.Run("featuring credits are left out of the lookup but kept in the title", func(t *testing.T) {
		f := &fakeFinder{album: Album{Title: "Album"}}
		got := correctAlbum(ctx, f, Tags{Artist: "A", Title: "Song (feat. B) (Live)"})
		require.Len(t, f.asked, 1)
		assert.Equal(t, "Song (Live)", f.asked[0].Title)
		assert.Equal(t, "Song (feat. B) (Live)", got.Title)
	})
}

func TestSuggestService_CorrectsAlbum(t *testing.T) {
	info := VideoInfo{Track: "Chop Suey!", Artists: []string{"System Of A Down"}, Album: "Chop Suey!"}
	svc := NewSuggestService(&fakeFetcher{info: info}, testDest)
	svc.SetAlbumFinder(&fakeFinder{album: Album{Title: "Toxicity", Year: 2001}})

	got, err := svc.Suggest(context.Background(), "https://www.youtube.com/v")

	require.NoError(t, err)
	assert.Equal(t, "Toxicity", got.Tags.Album)
	assert.Equal(t, 2001, got.Tags.Year)
}

func TestDownloadService_CorrectsAlbum(t *testing.T) {
	info := VideoInfo{Track: "Chop Suey!", Artists: []string{"System Of A Down"}, Album: "Chop Suey!"}
	video := Options{Type: MediaVideo, Format: "mp4"}

	download := func(t *testing.T, options Options, tags Tags) (FetchJob, *fakeFinder) {
		t.Helper()
		f := &fakeFetcher{info: info}
		finder := &fakeFinder{album: Album{Title: "Toxicity", Year: 2001}}
		svc := NewDownloadService(f, fixedDir{dir: t.TempDir()}, 1)
		svc.SetAlbumFinder(finder)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		req := validRequest
		req.Options = options
		req.Tags = tags
		d, err := svc.Start(req)
		require.NoError(t, err)
		statusesUntilTerminal(t, updates, d.ID)

		f.mu.Lock()
		defer f.mu.Unlock()
		require.Len(t, f.jobs, 1)
		return f.jobs[0], finder
	}

	t.Run("audio without a chosen album is corrected", func(t *testing.T) {
		job, _ := download(t, validRequest.Options, Tags{})
		assert.Equal(t, "Toxicity", job.Tags.Album)
		assert.Equal(t, []string{"System Of A Down", "Toxicity", "Chop Suey!"}, job.Path)
	})

	t.Run("an album the client chose is never looked up", func(t *testing.T) {
		job, finder := download(t, validRequest.Options, Tags{Album: "My Mix"})
		assert.Equal(t, "My Mix", job.Tags.Album)
		assert.Empty(t, finder.asked)
	})

	t.Run("video isn't looked up", func(t *testing.T) {
		job, finder := download(t, video, Tags{})
		assert.Equal(t, "Chop Suey!", job.Tags.Album)
		assert.Empty(t, finder.asked)
	})
}
