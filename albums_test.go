package gphotos

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gphotosuploader/googlemirror/api/photoslibrary/v1"

	"github.com/gphotosuploader/google-photos-api-client-go/v2/internal/cache"
)

type mockedCache struct{}

func (mc *mockedCache) GetAlbum(ctx context.Context, key string) (*photoslibrary.Album, error) {
	if key == "cached" {
		return &photoslibrary.Album{Title: "cached"}, nil
	}
	return nil, cache.ErrCacheMiss
}

func (mc *mockedCache) PutAlbum(ctx context.Context, key string, album *photoslibrary.Album, ttl time.Duration) error {
	return nil
}
func (mc *mockedCache) InvalidateAlbum(ctx context.Context, key string) error {
	return nil
}

// albumGallery represents the Album repository to mock Google Photos calls.
var albumGallery []*photoslibrary.Album

type mockedService struct{}

func (c *mockedService) ListAlbums(ctx context.Context, pageSize int64, pageToken string) (*photoslibrary.ListAlbumsResponse, error) {
	if pageToken == "give-me-more" {
		// second page of albums.
		return &photoslibrary.ListAlbumsResponse{
			Albums: albumGallery[pageSize:],
		}, nil
	}

	if pageSize < int64(len(albumGallery)) {
		// first page of albums.
		return &photoslibrary.ListAlbumsResponse{
			Albums: albumGallery[:pageSize],
			NextPageToken: "give-me-more",
		}, nil
	}

	// there is only one page of albums.
	return &photoslibrary.ListAlbumsResponse{
		Albums: albumGallery,
	}, nil
}

func (c *mockedService) CreateAlbum(ctx context.Context, request *photoslibrary.CreateAlbumRequest) (*photoslibrary.Album, error) {
	if request.Album.Title == "should-fail" {
		return nil, errors.New("album creation failure")
	}
	return request.Album, nil
}

func (c *mockedService) CreateMediaItems(ctx context.Context, request *photoslibrary.BatchCreateMediaItemsRequest) (*photoslibrary.BatchCreateMediaItemsResponse, error) {
	return &photoslibrary.BatchCreateMediaItemsResponse{}, nil
}

// initializeAlbumGallery will add the specified number of albums to the Album gallery.
// All the albums follow the template `album-<number>` where `<number>` is an incremental integer.
func initializeAlbumGallery(n int) {
	truncateAlbumGallery()
	for i := 1; i <= n; i++ {
		a := photoslibrary.Album{Title: fmt.Sprintf("album-%d", i)}
		albumGallery = append(albumGallery, &a)
	}
}

// truncateAlbumGallery will empty the Album gallery.
func truncateAlbumGallery() {
	albumGallery = nil
}

func TestClient_FindAlbum(t *testing.T) {
	ctx := context.Background()
	c := &Client{
		service: &mockedService{},
		cache:   &mockedCache{},
	}

	t.Run("WithNonExistentAlbum", func(t *testing.T) {
		truncateAlbumGallery()
		_, err := c.FindAlbum(ctx, "nonexistent")
		if err != ErrAlbumNotFound {
			t.Errorf("error was not expected. want: %v, got: %v", ErrAlbumNotFound, err)
		}
	})

	t.Run("WithCachedAlbum", func(t *testing.T) {
		truncateAlbumGallery()
		want := "cached"

		got, err := c.FindAlbum(ctx, want)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if want != got.Title {
			t.Errorf("want: %v, got: %v", want, got)
		}
	})

	t.Run("WithNonCachedAlbum", func(t *testing.T) {
		initializeAlbumGallery(5)
		want := "album-1"
		got, err := c.FindAlbum(ctx, want)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if want != got.Title {
			t.Errorf("want: %v, got: %v", want, got)
		}
	})
}

func TestClient_ListAlbums(t *testing.T) {
	ctx := context.Background()
	c := &Client{
		service: &mockedService{},
		cache:   &mockedCache{},
	}

	t.Run("WithEmptyAlbumGallery", func(t *testing.T) {
		truncateAlbumGallery()
		got, err := c.ListAlbums(ctx)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if len(got) > 0 {
			t.Errorf("no albums should be listed. got: %d", len(got))
		}
	})

	t.Run("WithSmallAlbumGallery", func(t *testing.T) {
		want := 5
		initializeAlbumGallery(want)
		got, err := c.ListAlbums(ctx)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if len(got) != want {
			t.Errorf("want: %d, got: %d", want, len(got))
		}
	})

	t.Run("WithLargeAlbumGallery", func(t *testing.T) {
		want := 500
		initializeAlbumGallery(want)
		got, err := c.ListAlbums(ctx)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if len(got) != want {
			t.Errorf("want: %d, got: %d", want, len(got))
		}
	})

}

func TestClient_CreateAlbum(t *testing.T) {
	ctx := context.Background()
	c := &Client{
		service: &mockedService{},
		cache:   &mockedCache{},
	}

	t.Run("ReturnsExistingAlbum", func(t *testing.T) {
		initializeAlbumGallery(1)
		want := "album-1"
		got, err := c.CreateAlbum(ctx, want)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if got.Title != want {
			t.Errorf("want: %s, got: %s", want, got.Title)
		}
	})

	t.Run("ReturnsCreatedAlbum", func(t *testing.T) {
		truncateAlbumGallery()
		want := "dummy"
		got, err := c.CreateAlbum(ctx, want)
		if err != nil {
			t.Fatalf("error was not expected at this point. err: %v", err)
		}
		if got.Title != want {
			t.Errorf("want: %s, got: %s", want, got.Title)
		}
	})

	t.Run("ShouldFailDueToAPIError", func(t *testing.T) {
		truncateAlbumGallery()
		want := "should-fail"
		got, err := c.CreateAlbum(ctx, want)
		if err == nil {
			t.Fatalf("error was expected at this point. got: %v", got)
		}
	})

}