package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCodeforcesUserNotFound = errors.New("codeforces user not found")
	ErrCodeforcesAPI          = errors.New("codeforces api error")
)

type CodeforcesClient struct {
	baseURL    string
	httpClient *http.Client
	now        func() time.Time
}

type CodeforcesProfile struct {
	Handle      string
	DisplayName string
	Rating      *int
	MaxRating   *int
	ProfileURL  string
	Payload     json.RawMessage
	FetchedAt   time.Time
}

type CodeforcesAcceptedSubmission struct {
	Handle       string
	ProblemKey   string
	ContestID    string
	ProblemIndex string
	ProblemName  string
	ProblemURL   string
	AcceptedAt   time.Time
	SubmissionID string
	SourceURL    string
	Payload      json.RawMessage
	FetchedAt    time.Time
}

type CodeforcesContestHistoryEntry struct {
	ContestID      string
	ContestName    string
	Rank           int
	OldRating      int
	NewRating      int
	RatingDelta    int
	ParticipatedAt time.Time
	SourceURL      string
	Payload        json.RawMessage
	FetchedAt      time.Time
}

func NewCodeforcesClient(baseURL string, timeout time.Duration) *CodeforcesClient {
	return &CodeforcesClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		now: time.Now,
	}
}

func (c *CodeforcesClient) FetchProfile(
	ctx context.Context,
	handle string,
) (CodeforcesProfile, error) {
	response, err := fetchCodeforcesAPI[[]codeforcesUserInfoResponse](
		ctx,
		c.httpClient,
		c.baseURL,
		"/user.info",
		url.Values{"handles": []string{handle}},
	)
	if err != nil {
		return CodeforcesProfile{}, err
	}
	if len(response.Result) == 0 {
		return CodeforcesProfile{}, fmt.Errorf("%w: empty user.info result", ErrCodeforcesAPI)
	}

	user := response.Result[0]
	payload, err := json.Marshal(user)
	if err != nil {
		return CodeforcesProfile{}, fmt.Errorf("marshal codeforces profile payload: %w", err)
	}

	return CodeforcesProfile{
		Handle:      user.Handle,
		DisplayName: user.Handle,
		Rating:      user.Rating,
		MaxRating:   user.MaxRating,
		ProfileURL:  fmt.Sprintf("https://codeforces.com/profile/%s", user.Handle),
		Payload:     payload,
		FetchedAt:   c.now().UTC(),
	}, nil
}

func (c *CodeforcesClient) FetchAcceptedSubmissions(
	ctx context.Context,
	handle string,
) ([]CodeforcesAcceptedSubmission, error) {
	response, err := fetchCodeforcesAPI[[]codeforcesSubmissionResponse](
		ctx,
		c.httpClient,
		c.baseURL,
		"/user.status",
		url.Values{"handle": []string{handle}},
	)
	if err != nil {
		return nil, err
	}

	fetchedAt := c.now().UTC()
	result := make([]CodeforcesAcceptedSubmission, 0, len(response.Result))
	for _, submission := range response.Result {
		if submission.Verdict != "OK" || submission.Problem.ContestID == nil || submission.Problem.Index == "" {
			continue
		}

		payload, err := json.Marshal(submission)
		if err != nil {
			return nil, fmt.Errorf("marshal codeforces submission payload: %w", err)
		}

		contestID := strconv.Itoa(*submission.Problem.ContestID)
		problemIndex := submission.Problem.Index
		submissionID := strconv.FormatInt(submission.ID, 10)
		result = append(result, CodeforcesAcceptedSubmission{
			Handle:       handle,
			ProblemKey:   fmt.Sprintf("CF-%s%s", contestID, problemIndex),
			ContestID:    contestID,
			ProblemIndex: problemIndex,
			ProblemName:  submission.Problem.Name,
			ProblemURL:   fmt.Sprintf("https://codeforces.com/contest/%s/problem/%s", contestID, problemIndex),
			AcceptedAt:   time.Unix(submission.CreationTimeSeconds, 0).UTC(),
			SubmissionID: submissionID,
			SourceURL:    fmt.Sprintf("https://codeforces.com/contest/%s/submission/%s", contestID, submissionID),
			Payload:      payload,
			FetchedAt:    fetchedAt,
		})
	}

	return result, nil
}

func (c *CodeforcesClient) FetchContestHistory(
	ctx context.Context,
	handle string,
) ([]CodeforcesContestHistoryEntry, error) {
	response, err := fetchCodeforcesAPI[[]codeforcesRatingChangeResponse](
		ctx,
		c.httpClient,
		c.baseURL,
		"/user.rating",
		url.Values{"handle": []string{handle}},
	)
	if err != nil {
		return nil, err
	}

	fetchedAt := c.now().UTC()
	result := make([]CodeforcesContestHistoryEntry, 0, len(response.Result))
	for _, contest := range response.Result {
		payload, err := json.Marshal(contest)
		if err != nil {
			return nil, fmt.Errorf("marshal codeforces contest payload: %w", err)
		}

		contestID := strconv.FormatInt(contest.ContestID, 10)
		result = append(result, CodeforcesContestHistoryEntry{
			ContestID:      contestID,
			ContestName:    contest.ContestName,
			Rank:           contest.Rank,
			OldRating:      contest.OldRating,
			NewRating:      contest.NewRating,
			RatingDelta:    contest.NewRating - contest.OldRating,
			ParticipatedAt: time.Unix(contest.RatingUpdateTimeSeconds, 0).UTC(),
			SourceURL:      fmt.Sprintf("https://codeforces.com/contest/%s", contestID),
			Payload:        payload,
			FetchedAt:      fetchedAt,
		})
	}

	return result, nil
}

type codeforcesAPIResponse[T any] struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
	Result  T      `json:"result"`
}

type codeforcesUserInfoResponse struct {
	Handle    string `json:"handle"`
	Rating    *int   `json:"rating"`
	MaxRating *int   `json:"maxRating"`
}

type codeforcesSubmissionResponse struct {
	ID                  int64  `json:"id"`
	CreationTimeSeconds int64  `json:"creationTimeSeconds"`
	Verdict             string `json:"verdict"`
	Problem             struct {
		ContestID *int   `json:"contestId"`
		Index     string `json:"index"`
		Name      string `json:"name"`
	} `json:"problem"`
}

type codeforcesRatingChangeResponse struct {
	ContestID               int64  `json:"contestId"`
	ContestName             string `json:"contestName"`
	Rank                    int    `json:"rank"`
	OldRating               int    `json:"oldRating"`
	NewRating               int    `json:"newRating"`
	RatingUpdateTimeSeconds int64  `json:"ratingUpdateTimeSeconds"`
}

func fetchCodeforcesAPI[T any](
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	query url.Values,
) (codeforcesAPIResponse[T], error) {
	var response codeforcesAPIResponse[T]

	endpoint := baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return response, fmt.Errorf("build codeforces request: %w", err)
	}

	httpResponse, err := client.Do(request)
	if err != nil {
		return response, fmt.Errorf("%w: request %s: %w", ErrCodeforcesAPI, endpoint, err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return response, fmt.Errorf("%w: request %s returned %s", ErrCodeforcesAPI, endpoint, httpResponse.Status)
	}

	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return response, fmt.Errorf("%w: decode %s: %w", ErrCodeforcesAPI, endpoint, err)
	}

	if response.Status != "OK" {
		if strings.Contains(strings.ToLower(response.Comment), "user with handle") {
			return response, ErrCodeforcesUserNotFound
		}

		return response, fmt.Errorf("%w: %s", ErrCodeforcesAPI, response.Comment)
	}

	return response, nil
}
