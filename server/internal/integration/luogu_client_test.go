package integration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLuoguClientFetchSnapshotMapsPublicUserData(t *testing.T) {
	t.Parallel()

	client := NewLuoguClient("https://www.luogu.com.cn", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/api/user/search":
				if r.URL.Query().Get("keyword") != "qiaochu" {
					t.Fatalf("search keyword = %q", r.URL.Query().Get("keyword"))
				}
				return jsonResponse(`{"users":[{"uid":809639,"name":"qiaochu"}]}`), nil
			case "/api/user/info/809639":
				return jsonResponse(`{"user":{"uid":809639,"name":"qiaochu","eloValue":1198}}`), nil
			case "/user/809639/practice":
				return htmlResponse(`<!DOCTYPE html><script id="lentille-context" type="application/json">{"data":{"passed":[{"type":"P","title":"A+B Problem","difficulty":1,"pid":"P1001"}]}}</script>`), nil
			default:
				t.Fatalf("unexpected request path = %q", r.URL.Path)
				return nil, nil
			}
		}),
	}
	client.now = func() time.Time { return time.Unix(1_700_700_000, 0).UTC() }

	snapshot, err := client.FetchSnapshot(context.Background(), "qiaochu")
	if err != nil {
		t.Fatalf("FetchSnapshot() error = %v", err)
	}

	if snapshot.Profile.UID != 809639 || snapshot.Profile.Handle != "qiaochu" {
		t.Fatalf("FetchSnapshot() profile = %+v", snapshot.Profile)
	}
	if snapshot.Profile.Rating == nil || *snapshot.Profile.Rating != 1198 {
		t.Fatalf("FetchSnapshot() rating = %#v", snapshot.Profile.Rating)
	}
	if len(snapshot.AcceptedProblems) != 1 {
		t.Fatalf("FetchSnapshot() accepted len = %d, want 1", len(snapshot.AcceptedProblems))
	}
	if snapshot.AcceptedProblems[0].ProblemKey != "P1001" {
		t.Fatalf("FetchSnapshot() problem key = %q, want %q", snapshot.AcceptedProblems[0].ProblemKey, "P1001")
	}
}

func TestLuoguClientFetchSnapshotMapsUserNotFound(t *testing.T) {
	t.Parallel()

	client := NewLuoguClient("https://www.luogu.com.cn", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(`{"users":[]}`), nil
		}),
	}

	_, err := client.FetchSnapshot(context.Background(), "missing")
	if !errors.Is(err, ErrLuoguUserNotFound) {
		t.Fatalf("FetchSnapshot() error = %v, want %v", err, ErrLuoguUserNotFound)
	}
}

func TestLuoguClientFetchSnapshotWrapsTransportError(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("dial failed")
	client := NewLuoguClient("https://www.luogu.com.cn", 3*time.Second)
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, transportErr
		}),
	}

	_, err := client.FetchSnapshot(context.Background(), "qiaochu")
	if !errors.Is(err, ErrLuoguAPI) {
		t.Fatalf("FetchSnapshot() error = %v, want wrapped %v", err, ErrLuoguAPI)
	}
	if !errors.Is(err, transportErr) {
		t.Fatalf("FetchSnapshot() error = %v, want wrapped transport error", err)
	}
}

func htmlResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
		Body: ioNopCloser(strings.NewReader(body)),
	}
}

type nopCloser struct {
	*strings.Reader
}

func (n nopCloser) Close() error {
	return nil
}

func ioNopCloser(reader *strings.Reader) nopCloser {
	return nopCloser{Reader: reader}
}
