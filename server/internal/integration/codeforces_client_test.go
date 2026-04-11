package integration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCodeforcesClientFetchProfileMapsUserInfo(t *testing.T) {
	t.Parallel()

	client := NewCodeforcesClient("https://cf.example.test", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/user.info" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/user.info")
			}

			return jsonResponse(`{"status":"OK","result":[{"handle":"tourist","rating":3700,"maxRating":3826}]}`), nil
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_300_000, 0).UTC() }

	profile, err := client.FetchProfile(context.Background(), "tourist")
	if err != nil {
		t.Fatalf("FetchProfile() error = %v", err)
	}

	if profile.Handle != "tourist" || profile.DisplayName != "tourist" {
		t.Fatalf("FetchProfile() profile = %+v", profile)
	}
	if profile.Rating == nil || *profile.Rating != 3700 {
		t.Fatalf("FetchProfile() rating = %#v", profile.Rating)
	}
	if profile.MaxRating == nil || *profile.MaxRating != 3826 {
		t.Fatalf("FetchProfile() max rating = %#v", profile.MaxRating)
	}
}

func TestCodeforcesClientFetchAcceptedSubmissionsFiltersAcceptedProblems(t *testing.T) {
	t.Parallel()

	client := NewCodeforcesClient("https://cf.example.test", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/user.status" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/user.status")
			}

			return jsonResponse(`{"status":"OK","result":[
{"id":10,"creationTimeSeconds":1700300000,"verdict":"WRONG_ANSWER","problem":{"contestId":1000,"index":"A","name":"Skip"}},
{"id":11,"creationTimeSeconds":1700300100,"verdict":"OK","problem":{"contestId":1000,"index":"A","name":"Accepted"}}
]}`), nil
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_300_500, 0).UTC() }

	submissions, err := client.FetchAcceptedSubmissions(context.Background(), "tourist")
	if err != nil {
		t.Fatalf("FetchAcceptedSubmissions() error = %v", err)
	}

	if len(submissions) != 1 {
		t.Fatalf("FetchAcceptedSubmissions() len = %d, want %d", len(submissions), 1)
	}
	if submissions[0].ProblemKey != "CF-1000A" {
		t.Fatalf("FetchAcceptedSubmissions() problem key = %q, want %q", submissions[0].ProblemKey, "CF-1000A")
	}
	if submissions[0].SubmissionID != "11" {
		t.Fatalf("FetchAcceptedSubmissions() submission id = %q, want %q", submissions[0].SubmissionID, "11")
	}
}

func TestCodeforcesClientFetchContestHistoryMapsRatingChanges(t *testing.T) {
	t.Parallel()

	client := NewCodeforcesClient("https://cf.example.test", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/user.rating" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/user.rating")
			}

			return jsonResponse(`{"status":"OK","result":[{"contestId":2000,"contestName":"Round","rank":12,"oldRating":1500,"newRating":1600,"ratingUpdateTimeSeconds":1700300200}]}`), nil
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_300_800, 0).UTC() }

	history, err := client.FetchContestHistory(context.Background(), "tourist")
	if err != nil {
		t.Fatalf("FetchContestHistory() error = %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("FetchContestHistory() len = %d, want %d", len(history), 1)
	}
	if history[0].RatingDelta != 100 {
		t.Fatalf("FetchContestHistory() rating delta = %d, want %d", history[0].RatingDelta, 100)
	}
}

func TestCodeforcesClientFetchProfileMapsUserNotFound(t *testing.T) {
	t.Parallel()

	client := NewCodeforcesClient("https://cf.example.test", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(`{"status":"FAILED","comment":"handles: User with handle missing not found"}`), nil
		}),
	}

	if _, err := client.FetchProfile(context.Background(), "missing"); err != ErrCodeforcesUserNotFound {
		t.Fatalf("FetchProfile() error = %v, want %v", err, ErrCodeforcesUserNotFound)
	}
}

func TestCodeforcesClientFetchProfileWrapsUnderlyingTransportError(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("dial failed")
	client := NewCodeforcesClient("https://cf.example.test", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, transportErr
		}),
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("FetchProfile() panicked: %v", recovered)
		}
	}()

	_, err := client.FetchProfile(context.Background(), "tourist")
	if !errors.Is(err, ErrCodeforcesAPI) {
		t.Fatalf("FetchProfile() error = %v, want wrapped %v", err, ErrCodeforcesAPI)
	}
	if !errors.Is(err, transportErr) {
		t.Fatalf("FetchProfile() error = %v, want wrapped transport error", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
