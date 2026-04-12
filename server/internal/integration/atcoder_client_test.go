package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAtCoderClientFetchProfileParsesAlgorithmRating(t *testing.T) {
	t.Parallel()

	client := NewAtCoderClient("https://atcoder.example.test", 3*time.Second, nil)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/users/tourist" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/users/tourist")
			}

			return htmlResponse(`
<html><body>
<table class="dl-table mt-2">
  <tr><th class="no-break">Rating</th><td><span class='user-red'>3797</span></td></tr>
  <tr><th class="no-break">Highest Rating</th><td><span class='user-red'>4229</span></td></tr>
</table>
</body></html>`), nil
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_910_000, 0).UTC() }

	profile, err := client.FetchProfile(context.Background(), "tourist")
	if err != nil {
		t.Fatalf("FetchProfile() error = %v", err)
	}

	if profile.Handle != "tourist" || profile.DisplayName != "tourist" {
		t.Fatalf("FetchProfile() profile = %+v", profile)
	}
	if profile.Rating == nil || *profile.Rating != 3797 {
		t.Fatalf("FetchProfile() rating = %#v", profile.Rating)
	}
	if profile.MaxRating == nil || *profile.MaxRating != 4229 {
		t.Fatalf("FetchProfile() max rating = %#v", profile.MaxRating)
	}
}

func TestAtCoderClientFetchContestHistoryMapsOfficialJSON(t *testing.T) {
	t.Parallel()

	client := NewAtCoderClient("https://atcoder.example.test", 3*time.Second, nil)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/users/tourist/history/json" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/users/tourist/history/json")
			}

			return jsonResponse(`[{
  "IsRated": true,
  "Place": 1,
  "OldRating": 3800,
  "NewRating": 3830,
  "ContestScreenName": "agc077.contest.atcoder.jp",
  "ContestName": "AtCoder Grand Contest 077",
  "ContestNameEn": "",
  "EndTime": "2026-03-30T00:00:00+09:00"
}]`), nil
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_910_100, 0).UTC() }

	history, err := client.FetchContestHistory(context.Background(), "tourist")
	if err != nil {
		t.Fatalf("FetchContestHistory() error = %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("FetchContestHistory() len = %d, want 1", len(history))
	}
	if history[0].ContestID != "agc077" {
		t.Fatalf("FetchContestHistory() contest id = %q, want %q", history[0].ContestID, "agc077")
	}
	if history[0].RatingDelta == nil || *history[0].RatingDelta != 30 {
		t.Fatalf("FetchContestHistory() rating delta = %#v", history[0].RatingDelta)
	}
}

func TestAtCoderClientFetchAcceptedSubmissionsUsesCookieAndFiltersAccepted(t *testing.T) {
	t.Parallel()

	client := NewAtCoderClient(
		"https://atcoder.example.test",
		3*time.Second,
		func(context.Context) (string, error) {
			return "REVEL_SESSION=test", nil
		},
	)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Cookie"); got != "REVEL_SESSION=test" {
				t.Fatalf("Cookie header = %q, want %q", got, "REVEL_SESSION=test")
			}

			if r.URL.Path != "/contests/agc077/submissions" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/contests/agc077/submissions")
			}

			switch r.URL.Query().Get("page") {
			case "1":
				return htmlResponse(`
<html><body>
<table><tbody>
  <tr>
    <td><time>2026-03-30 00:10:00+0900</time></td>
    <td><a href="/contests/agc077/tasks/agc077_a">A - Candy</a></td>
    <td><span class="label label-success">AC</span></td>
    <td><a href="/contests/agc077/submissions/6001">Detail</a></td>
  </tr>
  <tr>
    <td><time>2026-03-30 00:20:00+0900</time></td>
    <td><a href="/contests/agc077/tasks/agc077_b">B - Skip</a></td>
    <td><span class="label label-warning">WA</span></td>
    <td><a href="/contests/agc077/submissions/6002">Detail</a></td>
  </tr>
</tbody></table>
</body></html>`), nil
			case "2":
				return htmlResponse(`<html><body><table><tbody></tbody></table></body></html>`), nil
			default:
				t.Fatalf("unexpected page query = %q", r.URL.Query().Get("page"))
				return nil, nil
			}
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_910_200, 0).UTC() }

	submissions, err := client.FetchAcceptedSubmissions(context.Background(), "tourist", []string{"agc077"})
	if err != nil {
		t.Fatalf("FetchAcceptedSubmissions() error = %v", err)
	}

	if len(submissions) != 1 {
		t.Fatalf("FetchAcceptedSubmissions() len = %d, want 1", len(submissions))
	}
	if submissions[0].ProblemKey != "agc077_a" {
		t.Fatalf("FetchAcceptedSubmissions() problem key = %q, want %q", submissions[0].ProblemKey, "agc077_a")
	}
	if submissions[0].SubmissionID != "6001" {
		t.Fatalf("FetchAcceptedSubmissions() submission id = %q, want %q", submissions[0].SubmissionID, "6001")
	}
}

func TestAtCoderClientCheckReadyRequiresCookieHeader(t *testing.T) {
	t.Parallel()

	client := NewAtCoderClient(
		"https://atcoder.example.test",
		3*time.Second,
		func(context.Context) (string, error) {
			return "", nil
		},
	)

	err := client.CheckReady(context.Background())
	if !errors.Is(err, ErrAtCoderCredentialMissing) {
		t.Fatalf("CheckReady() error = %v, want %v", err, ErrAtCoderCredentialMissing)
	}
}
