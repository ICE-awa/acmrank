package integration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestICPCAwardClientFetchAwardsParsesEnvelope(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_000, 0).UTC()
	client.now = func() time.Time { return now }
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/awards.json" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/awards.json")
		}

		return jsonHTTPResponse(http.StatusOK, `{
  "records": [
    {
      "contest_name": "ICPC Asia Regional 2025",
      "award_name": "Gold Medal",
      "rank_text": "Rank 3",
      "award_date": "2025-11-02",
      "members": ["Alice", " Bob "],
      "source_url": "https://board.example.test/regional-2025"
    }
  ]
}`), nil
	})
	records, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("FetchAwards() len = %d, want 1", len(records))
	}

	record := records[0]
	if record.ContestName != "ICPC Asia Regional 2025" {
		t.Fatalf("ContestName = %q", record.ContestName)
	}

	if len(record.Members) != 2 || record.Members[1] != "Bob" {
		t.Fatalf("Members = %#v", record.Members)
	}
	if len(record.NormalizedMembers) != 2 || record.NormalizedMembers[1] != "bob" {
		t.Fatalf("NormalizedMembers = %#v", record.NormalizedMembers)
	}

	if got := record.AwardDate.Format(time.DateOnly); got != "2025-11-02" {
		t.Fatalf("AwardDate = %q, want %q", got, "2025-11-02")
	}
}

func TestICPCAwardClientFetchAwardsFallsBackToArrayPayload(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_100, 0).UTC()
	client.now = func() time.Time { return now }
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `[
  {
    "contest_name": "ICPC EC Final 2024",
    "award_name": "Silver Medal",
    "award_date": "2024-12-01",
    "members": ["Carol"]
  }
]`), nil
	})
	records, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("FetchAwards() len = %d, want 1", len(records))
	}

	if records[0].SourceURL != "https://icpc.example.test/awards.json" {
		t.Fatalf("SourceURL = %q, want %q", records[0].SourceURL, "https://icpc.example.test/awards.json")
	}
}

func TestICPCAwardClientFetchAwardsRejectsInvalidResponse(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_200, 0).UTC()
	client.now = func() time.Time { return now }
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusBadGateway, "boom"), nil
	})
	_, err := client.FetchAwards(context.Background())
	if !errors.Is(err, ErrICPCAwardsAPI) {
		t.Fatalf("FetchAwards() error = %v, want %v", err, ErrICPCAwardsAPI)
	}
}

func TestICPCAwardClientFetchAwardsRejectsOversizedFeed(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_300, 0).UTC()
	client.now = func() time.Time { return now }
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, strings.Repeat(" ", maxICPCAwardFeedBytes+1)), nil
	})

	_, err := client.FetchAwards(context.Background())
	if !errors.Is(err, ErrICPCAwardsAPI) {
		t.Fatalf("FetchAwards() error = %v, want %v", err, ErrICPCAwardsAPI)
	}
}

func TestICPCAwardClientFetchAwardsSkipsMalformedRecords(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_400, 0).UTC()
	client.now = func() time.Time { return now }
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `{
  "records": [
    {
      "contest_name": "Broken Regional",
      "award_name": "Gold Medal",
      "award_date": "2025/11/02",
      "members": ["Alice"]
    },
    {
      "contest_name": "ICPC Asia Regional 2025",
      "award_name": "Silver Medal",
      "award_date": "2025-11-02",
      "members": ["Bob"]
    }
  ]
}`), nil
	})

	records, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards() error = %v", err)
	}

	if len(records) != 1 || records[0].ContestName != "ICPC Asia Regional 2025" {
		t.Fatalf("FetchAwards() records = %#v", records)
	}
}

func TestICPCAwardClientFetchAwardsUsesInMemoryCache(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_500, 0).UTC()
	client.now = func() time.Time { return now }

	requestCount := 0
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestCount++
		return jsonHTTPResponse(http.StatusOK, `{
  "records": [
    {
      "contest_name": "ICPC Asia Regional 2025",
      "award_name": "Gold Medal",
      "award_date": "2025-11-02",
      "members": ["Alice"]
    }
  ]
}`), nil
	})

	records, err := client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards() first call error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("FetchAwards() first call len = %d, want 1", len(records))
	}

	records, err = client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards() second call error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("FetchAwards() second call len = %d, want 1", len(records))
	}

	if requestCount != 1 {
		t.Fatalf("requestCount after cached call = %d, want 1", requestCount)
	}

	now = now.Add(icpcAwardCacheTTL + time.Second)
	records, err = client.FetchAwards(context.Background())
	if err != nil {
		t.Fatalf("FetchAwards() third call error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("FetchAwards() third call len = %d, want 1", len(records))
	}

	if requestCount != 2 {
		t.Fatalf("requestCount after cache expiry = %d, want 2", requestCount)
	}
}

func TestICPCAwardClientFetchAwardsSharesConcurrentFetch(t *testing.T) {
	t.Parallel()

	client := NewICPCAwardClient("https://icpc.example.test/awards.json", 5*time.Second)
	now := time.Unix(1_701_200_600, 0).UTC()
	client.now = func() time.Time { return now }

	var requestCount atomic.Int32
	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	client.httpClient.Transport = icpcRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestCount.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release

		return jsonHTTPResponse(http.StatusOK, `{
  "records": [
    {
      "contest_name": "ICPC Asia Regional 2025",
      "award_name": "Gold Medal",
      "award_date": "2025-11-02",
      "members": ["Alice"]
    }
  ]
}`), nil
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			records, err := client.FetchAwards(context.Background())
			if err != nil {
				errCh <- err
				return
			}
			if len(records) != 1 {
				errCh <- errors.New("unexpected record count")
			}
		}()
	}

	<-entered
	time.Sleep(50 * time.Millisecond)
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("requestCount while fetch in flight = %d, want 1", got)
	}

	close(release)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("FetchAwards() concurrent error = %v", err)
		}
	}
}

type icpcRoundTripFunc func(*http.Request) (*http.Response, error)

func (f icpcRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
