package ytdlp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearcher_SearchMusic(t *testing.T) {
	// The search prints three IDs; looking up "bad" fails.
	s := &Searcher{Executable: fakeExecutable(t, `
for a in "$@"; do last="$a"; done
case " $* " in
*" --flat-playlist "*)
  case "$last" in "https://music.youtube.com/search?q=queen+bohemian#songs") ;; *) echo "ERROR: bad url $last" >&2; exit 1;; esac
  printf 'a1\nbad\nb2\n';;
*watch?v=a1*)
  echo '{"id":"a1","title":"Bohemian Rhapsody","track":"Bohemian Rhapsody","artists":["Queen"],"album":"A Night At The Opera","release_date":"19751121","duration":354,"thumbnail":"https://i.ytimg.com/a1.jpg"}';;
*watch?v=b2*)
  sleep 0.2  # finishes last: order must still follow the search
  echo '{"id":"b2","title":"Bohemian Rhapsody","track":"Bohemian Rhapsody","artists":["Alex Goot","Sam Tsui"],"album":"Bohemian Rhapsody","duration":356}';;
*) echo "ERROR: Video unavailable" >&2; exit 1;;
esac`)}

	got, err := s.SearchMusic(context.Background(), "queen bohemian", 3)
	require.NoError(t, err)
	require.Len(t, got, 2, "the failed lookup is skipped")

	assert.Equal(t, media.Video{
		ID: "a1", Title: "Bohemian Rhapsody", Uploader: "Queen",
		Duration:   354 * time.Second,
		Thumbnails: []media.Thumbnail{{URL: "https://i.ytimg.com/a1.jpg"}},
		URL:        "https://music.youtube.com/watch?v=a1",
		Album:      "A Night At The Opera",
		Tags:       &media.Tags{Artist: "Queen", Title: "Bohemian Rhapsody", Album: "A Night At The Opera", Year: 1975},
	}, got[0])

	assert.Equal(t, "b2", got[1].ID)
	assert.Equal(t, "Alex Goot, Sam Tsui", got[1].Uploader)
	assert.Equal(t, "Bohemian Rhapsody (feat. Sam Tsui)", got[1].Tags.Title)
}

func TestSearcher_SearchMusic_AllLookupsFail(t *testing.T) {
	s := &Searcher{Executable: fakeExecutable(t, `
case " $* " in *" --flat-playlist "*) printf 'a\nb\n';; *) echo 'ERROR: Sign in to confirm you are not a bot' >&2; exit 1;; esac`)}
	_, err := s.SearchMusic(context.Background(), "q", 2)
	assert.ErrorIs(t, err, media.ErrSearchFailed)
	assert.EqualError(t, err, "search failed: Sign in to confirm you are not a bot")
}

func TestErrorMessage(t *testing.T) {
	tests := map[string]string{
		"exit code 1: exit status 1\n\nERROR: [youtube] BSTsnWoslP4: Sign in to confirm you’re not a bot. This helps protect our community. See https://x": "Sign in to confirm you’re not a bot.",
		"ERROR: first\nERROR: [youtube:search] HTTP Error 429: Too Many Requests":                                                                          "HTTP Error 429: Too Many Requests",
		"ERROR: Unable to download webpage: timed out":                                                                                                     "Unable to download webpage: timed out",
		"exec: yt-dlp: not found": "exec: yt-dlp: not found",
	}
	for in, want := range tests {
		assert.Equal(t, want, errorMessage(errors.New(in)), in)
	}
}

func TestSearcher_SearchMusic_SearchFails(t *testing.T) {
	s := &Searcher{Executable: fakeExecutable(t, `echo 'ERROR: HTTP Error 429' >&2; exit 1`)}
	_, err := s.SearchMusic(context.Background(), "q", 5)
	assert.Error(t, err)
}

func TestSearcher_SearchMusic_Cancelled(t *testing.T) {
	s := &Searcher{Executable: fakeExecutable(t, `
case " $* " in *" --flat-playlist "*) printf 'a\nb\n';; *) sleep 5;; esac`)}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := s.SearchMusic(ctx, "q", 2)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 3*time.Second, "lookups must stop with the request")
}
