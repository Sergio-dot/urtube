package musicbrainz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func group(id, title, primary string, secondary ...string) releaseGroup {
	return releaseGroup{ID: id, Title: title, PrimaryType: primary, SecondaryTypes: secondary}
}

func official(date string, g releaseGroup) release {
	return release{Date: date, Status: "Official", ReleaseGroup: g}
}

func song(artist, title string, releases ...release) recording {
	r := recording{Title: title, Releases: releases}
	r.ArtistCredit = []credit{{Name: artist}}
	return r
}

var (
	single      = group("single", "Chop Suey!", "Single")
	toxicity    = group("toxicity", "Toxicity", "Album")
	stealThis   = group("steal", "Steal This Album!", "Album")
	greatest    = group("greatest", "Greatest Hits", "Album", "Compilation")
	liveAlbum   = group("live", "Pledge of Allegiance Tour", "Album", "Live")
	systemTags  = media.Tags{Artist: "System Of A Down", Title: "Chop Suey!", Album: "Chop Suey!"}
	systemMixed = "System of a Down"
)

func TestPickAlbum(t *testing.T) {
	tests := []struct {
		name       string
		tags       media.Tags
		recordings []recording
		want       media.Album
	}{
		{
			name: "a single is corrected to its studio album",
			tags: systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!",
				official("2001-08-13", single), official("2001-09-04", toxicity), official("2001-09-04", toxicity))},
			want: media.Album{Title: "Toxicity"},
		},
		{
			name: "compilations and live albums don't count",
			tags: systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!",
				official("1999", greatest), official("2000", liveAlbum), official("2001-09-04", toxicity))},
			want: media.Album{Title: "Toxicity"},
		},
		{
			name:       "a song only released as a single has no album",
			tags:       systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!", official("2001-08-13", single))},
		},
		{
			name: "an album that is already a studio album is kept",
			tags: media.Tags{Artist: "System Of A Down", Title: "Innervision", Album: "steal this album"},
			recordings: []recording{song(systemMixed, "Innervision",
				official("1998", toxicity), official("2002", stealThis))},
			want: media.Album{Title: "steal this album"},
		},
		{
			name: "the earliest album wins",
			tags: systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!",
				official("2002", stealThis), official("2001", toxicity))},
			want: media.Album{Title: "Toxicity"},
		},
		{
			name: "albums with no date are ignored",
			tags: systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!",
				official("", stealThis), official("2001", toxicity))},
			want: media.Album{Title: "Toxicity"},
		},
		{
			name:       "only albums with no date means no album",
			tags:       systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!", official("", toxicity))},
		},
		{
			name:       "another song's album is ignored",
			tags:       systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey (remix)", official("2001", toxicity))},
		},
		{
			name:       "another artist's album is ignored",
			tags:       systemTags,
			recordings: []recording{song("A Tribute Band", "Chop Suey!", official("2001", toxicity))},
		},
		{
			name: "unofficial releases are ignored",
			tags: systemTags,
			recordings: []recording{song(systemMixed, "Chop Suey!",
				release{Date: "2001", Status: "Bootleg", ReleaseGroup: toxicity})},
		},
		{
			name: "the main artist of a joint credit matches",
			tags: media.Tags{Artist: "Queen", Title: "Under Pressure"},
			recordings: []recording{{
				Title:        "Under Pressure",
				ArtistCredit: []credit{{Name: "Queen", JoinPhrase: " & "}, {Name: "David Bowie"}},
				Releases:     []release{official("1982", group("hot", "Hot Space", "Album"))},
			}},
			want: media.Album{Title: "Hot Space"},
		},
		{
			name:       "a title of only symbols matches nothing",
			tags:       media.Tags{Artist: "X", Title: "???"},
			recordings: []recording{song("X", "!!!", official("2001", toxicity))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := pickAlbum(tt.recordings, tt.tags)
			assert.Equal(t, tt.want, got)
		})
	}
}

// fakeServer answers each request with handle.
func fakeServer(t *testing.T, handle http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handle)
	t.Cleanup(srv.Close)
	return &Client{
		baseURL:   srv.URL,
		http:      srv.Client(),
		interval:  time.Millisecond,
		retryWait: time.Millisecond,
		cooldown:  time.Minute,
		cache:     make(map[string]media.Album),
	}
}

func writeRecordings(t *testing.T, w http.ResponseWriter, recordings ...recording) {
	t.Helper()
	assert.NoError(t, json.NewEncoder(w).Encode(searchResponse{Recordings: recordings}))
}

func TestFindAlbum_Request(t *testing.T) {
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/release-group/toxicity" {
			_, _ = w.Write([]byte(`{"id": "toxicity", "first-release-date": "2001-09-04"}`))
			return
		}
		assert.Equal(t, "/recording", r.URL.Path)
		assert.Contains(t, r.Header.Get("User-Agent"), "urtube/")
		assert.Equal(t, "json", r.URL.Query().Get("fmt"))
		assert.Equal(t, "100", r.URL.Query().Get("limit"), "smaller limits miss the original album of popular songs")
		assert.Equal(t, `recording:"Say \"Hi\" \\ Bye" AND artist:"X" AND status:official`, r.URL.Query().Get("query"))
		writeRecordings(t, w, song("X", `Say "Hi" \ Bye`, official("1999", toxicity)))
	})

	album, err := c.FindAlbum(context.Background(), media.Tags{Artist: "X", Title: `Say "Hi" \ Bye`})

	require.NoError(t, err)
	assert.Equal(t, media.Album{Title: "Toxicity", Year: 2001}, album, "the year comes from the album's own record")
}

func TestFindAlbum_YearLookup(t *testing.T) {
	ctx := context.Background()
	search := []recording{song("X", "Y", official("2001", toxicity))}

	t.Run("a failed year lookup still gives the album", func(t *testing.T) {
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/recording" {
				writeRecordings(t, w, search...)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		})

		album, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})

		require.NoError(t, err)
		assert.Equal(t, media.Album{Title: "Toxicity"}, album)
	})

	t.Run("an album that isn't changing needs no year lookup", func(t *testing.T) {
		var calls atomic.Int32
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			writeRecordings(t, w, search...)
		})

		album, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y", Album: "Toxicity"})

		require.NoError(t, err)
		assert.Equal(t, media.Album{Title: "Toxicity"}, album)
		assert.EqualValues(t, 1, calls.Load())
	})
}

func TestFindAlbum_Retry(t *testing.T) {
	ctx := context.Background()

	t.Run("throttled once, then answered", func(t *testing.T) {
		var calls atomic.Int32
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writeRecordings(t, w, song("X", "Y", official("2001", toxicity)))
		})

		album, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})

		require.NoError(t, err)
		assert.Equal(t, "Toxicity", album.Title)
		assert.EqualValues(t, 3, calls.Load(), "the throttled search, its retry and the year lookup")
	})

	t.Run("retries only once", func(t *testing.T) {
		var calls atomic.Int32
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusTooManyRequests)
		})

		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})

		assert.ErrorIs(t, err, ErrThrottled)
		assert.EqualValues(t, 2, calls.Load())
	})

	t.Run("gives up when asked to wait too long, and pauses lookups", func(t *testing.T) {
		var calls atomic.Int32
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})
		assert.ErrorIs(t, err, ErrThrottled)
		assert.EqualValues(t, 1, calls.Load())

		_, err = c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Other"})
		assert.ErrorIs(t, err, ErrThrottled)
		assert.EqualValues(t, 1, calls.Load(), "nothing is sent during the pause")
	})

	t.Run("waits before retrying", func(t *testing.T) {
		const wait = 60 * time.Millisecond
		var (
			mu    sync.Mutex
			times []time.Time
		)
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			times = append(times, time.Now())
			first := len(times) == 1
			mu.Unlock()
			if first {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writeRecordings(t, w)
		})
		c.retryWait = wait

		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})

		require.NoError(t, err)
		require.Len(t, times, 2)
		assert.GreaterOrEqual(t, times[1].Sub(times[0]), wait-10*time.Millisecond)
	})

	t.Run("repeated throttling pauses every lookup until the cooldown ends", func(t *testing.T) {
		var calls, throttle atomic.Int32
		throttle.Store(1)
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if throttle.Load() == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writeRecordings(t, w)
		})
		c.cooldown = 80 * time.Millisecond

		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})
		assert.ErrorIs(t, err, ErrThrottled)
		assert.EqualValues(t, 2, calls.Load())

		start := time.Now()
		_, err = c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Other"})
		assert.ErrorIs(t, err, ErrThrottled)
		assert.Less(t, time.Since(start), 40*time.Millisecond, "fails at once")
		assert.EqualValues(t, 2, calls.Load(), "nothing is sent during the pause")

		throttle.Store(0)
		time.Sleep(100 * time.Millisecond)
		_, err = c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Other"})
		assert.NoError(t, err, "lookups resume after the cooldown")
		assert.EqualValues(t, 3, calls.Load())
	})

	t.Run("other failures aren't retried", func(t *testing.T) {
		var calls atomic.Int32
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})

		assert.ErrorContains(t, err, "500")
		assert.EqualValues(t, 1, calls.Load())
	})

	t.Run("a bad response is an error", func(t *testing.T) {
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("<", 10)))
		})
		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Y"})
		assert.ErrorContains(t, err, "parse MusicBrainz response")
	})
}

func TestDefaults(t *testing.T) {
	c := New()
	assert.Equal(t, 2*time.Second, c.interval)
	assert.Equal(t, 5*time.Second, c.retryWait)
	assert.Equal(t, 5*time.Minute, c.cooldown)
}

func TestCache(t *testing.T) {
	ctx := context.Background()
	tags := media.Tags{Artist: "System Of A Down", Title: "Chop Suey!", Album: "Chop Suey!"}

	countingServer := func(t *testing.T, status *atomic.Int32) (*Client, *atomic.Int32) {
		t.Helper()
		var searches atomic.Int32
		return fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/recording" {
				_, _ = w.Write([]byte(`{"first-release-date": "2001-09-04"}`))
				return
			}
			searches.Add(1)
			if status != nil && status.Load() != 0 {
				w.WriteHeader(int(status.Load()))
				return
			}
			writeRecordings(t, w, song(systemMixed, "Chop Suey!", official("2001", toxicity)))
		}), &searches
	}

	t.Run("the same song is only looked up once", func(t *testing.T) {
		c, searches := countingServer(t, nil)

		first, err := c.FindAlbum(ctx, tags)
		require.NoError(t, err)
		respelled := media.Tags{Artist: "system of a down", Title: "CHOP SUEY", Album: "chop suey"}
		second, err := c.FindAlbum(ctx, respelled)
		require.NoError(t, err)

		assert.Equal(t, media.Album{Title: "Toxicity", Year: 2001}, first)
		assert.Equal(t, first, second)
		assert.EqualValues(t, 1, searches.Load())
	})

	t.Run("no match is remembered too", func(t *testing.T) {
		c, searches := countingServer(t, nil)
		unknown := media.Tags{Artist: "Nobody", Title: "Nothing"}
		for range 2 {
			album, err := c.FindAlbum(ctx, unknown)
			require.NoError(t, err)
			assert.Zero(t, album)
		}
		assert.EqualValues(t, 1, searches.Load())
	})

	t.Run("a different album is a different question", func(t *testing.T) {
		c, searches := countingServer(t, nil)
		_, err := c.FindAlbum(ctx, tags)
		require.NoError(t, err)
		other := tags
		other.Album = "Toxicity"
		album, err := c.FindAlbum(ctx, other)
		require.NoError(t, err)
		assert.Equal(t, "Toxicity", album.Title)
		assert.EqualValues(t, 2, searches.Load())
	})

	t.Run("failures aren't remembered", func(t *testing.T) {
		var status atomic.Int32
		status.Store(http.StatusInternalServerError)
		c, _ := countingServer(t, &status)
		_, err := c.FindAlbum(ctx, tags)
		require.Error(t, err)

		status.Store(0)
		album, err := c.FindAlbum(ctx, tags)
		require.NoError(t, err)
		assert.Equal(t, "Toxicity", album.Title)
	})

	t.Run("an album without its year isn't remembered", func(t *testing.T) {
		var yearCalls atomic.Int32
		c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/recording" {
				writeRecordings(t, w, song(systemMixed, "Chop Suey!", official("2001", toxicity)))
				return
			}
			if yearCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"first-release-date": "2001-09-04"}`))
		})

		first, err := c.FindAlbum(ctx, tags)
		require.NoError(t, err)
		second, err := c.FindAlbum(ctx, tags)
		require.NoError(t, err)

		assert.Equal(t, 0, first.Year)
		assert.Equal(t, 2001, second.Year)
	})

	t.Run("the cache is bounded", func(t *testing.T) {
		c, _ := countingServer(t, nil)
		for i := range maxCached {
			c.remember(strconv.Itoa(i), media.Album{})
		}
		c.remember("one more", media.Album{Title: "Kept"})
		assert.Len(t, c.cache, 1)
	})
}

func TestRateLimit(t *testing.T) {
	const interval = 40 * time.Millisecond
	var (
		mu    sync.Mutex
		times []time.Time
	)
	c := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		times = append(times, time.Now())
		mu.Unlock()
		writeRecordings(t, w)
	})
	c.interval = interval

	t.Run("concurrent lookups are spaced apart", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range 4 {
			wg.Go(func() {
				_, err := c.FindAlbum(context.Background(), media.Tags{Artist: "X", Title: "Song " + strconv.Itoa(i)})
				assert.NoError(t, err)
			})
		}
		wg.Wait()

		require.Len(t, times, 4)
		for i := 1; i < len(times); i++ {
			// Allows for scheduling delays between the slot and the handler.
			assert.GreaterOrEqual(t, times[i].Sub(times[0]), time.Duration(i)*interval-10*time.Millisecond)
		}
	})

	t.Run("a lookup whose turn is past its deadline gives up at once", func(t *testing.T) {
		c.backOff(time.Minute)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		start := time.Now()
		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Busy"})

		assert.ErrorIs(t, err, ErrBusy)
		assert.Less(t, time.Since(start), 500*time.Millisecond)
	})

	t.Run("a cancelled lookup stops waiting", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := c.FindAlbum(ctx, media.Tags{Artist: "X", Title: "Cancelled"})
		assert.ErrorIs(t, err, context.Canceled)
	})
}
