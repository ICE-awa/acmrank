package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrLuoguUserNotFound = errors.New("luogu user not found")
	ErrLuoguAPI          = errors.New("luogu api error")
)

type LuoguClient struct {
	baseURL    string
	httpClient *http.Client
	now        func() time.Time
}

type LuoguProfile struct {
	UID         int64
	Handle      string
	DisplayName string
	Rating      *int
	ProfileURL  string
	Payload     json.RawMessage
	FetchedAt   time.Time
}

type LuoguAcceptedProblem struct {
	UID         int64
	Handle      string
	ProblemKey  string
	ProblemID   string
	ProblemName string
	ProblemURL  string
	SourceURL   string
	Payload     json.RawMessage
	FetchedAt   time.Time
}

type LuoguSyncSnapshot struct {
	Profile          LuoguProfile
	AcceptedProblems []LuoguAcceptedProblem
}

func NewLuoguClient(baseURL string, timeout time.Duration) *LuoguClient {
	return &LuoguClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		now: time.Now,
	}
}

func (c *LuoguClient) FetchSnapshot(
	ctx context.Context,
	handle string,
) (LuoguSyncSnapshot, error) {
	user, err := c.searchUser(ctx, handle)
	if err != nil {
		return LuoguSyncSnapshot{}, err
	}

	userInfo, err := c.fetchUserInfo(ctx, user.UID)
	if err != nil {
		return LuoguSyncSnapshot{}, err
	}

	practicePage, err := c.fetchPracticePage(ctx, user.UID)
	if err != nil {
		return LuoguSyncSnapshot{}, err
	}

	fetchedAt := c.now().UTC()
	profilePayload, err := json.Marshal(userInfo.User)
	if err != nil {
		return LuoguSyncSnapshot{}, fmt.Errorf("marshal luogu profile payload: %w", err)
	}

	snapshot := LuoguSyncSnapshot{
		Profile: LuoguProfile{
			UID:         user.UID,
			Handle:      user.Name,
			DisplayName: userInfo.User.Name,
			Rating:      userInfo.User.EloValue,
			ProfileURL:  fmt.Sprintf("%s/user/%d", c.baseURL, user.UID),
			Payload:     profilePayload,
			FetchedAt:   fetchedAt,
		},
		AcceptedProblems: make([]LuoguAcceptedProblem, 0, len(practicePage.Data.Passed)),
	}

	practiceURL := fmt.Sprintf("%s/user/%d/practice", c.baseURL, user.UID)
	for _, problem := range practicePage.Data.Passed {
		payload, err := json.Marshal(problem)
		if err != nil {
			return LuoguSyncSnapshot{}, fmt.Errorf("marshal luogu practice payload: %w", err)
		}

		snapshot.AcceptedProblems = append(snapshot.AcceptedProblems, LuoguAcceptedProblem{
			UID:         user.UID,
			Handle:      user.Name,
			ProblemKey:  problem.PID,
			ProblemID:   problem.PID,
			ProblemName: problem.Title,
			ProblemURL:  fmt.Sprintf("%s/problem/%s", c.baseURL, problem.PID),
			SourceURL:   practiceURL,
			Payload:     payload,
			FetchedAt:   fetchedAt,
		})
	}

	return snapshot, nil
}

type luoguSearchUserResponse struct {
	Users []struct {
		UID  int64  `json:"uid"`
		Name string `json:"name"`
	} `json:"users"`
}

type luoguUserInfoResponse struct {
	User struct {
		UID       int64  `json:"uid"`
		Name      string `json:"name"`
		EloValue  *int   `json:"eloValue"`
		Avatar    string `json:"avatar"`
		Slogan    string `json:"slogan"`
		Color     string `json:"color"`
		CCFLevel  int    `json:"ccfLevel"`
		XCPCLevel int    `json:"xcpcLevel"`
	} `json:"user"`
}

type luoguPracticePageResponse struct {
	Data struct {
		Passed []struct {
			Type       string `json:"type"`
			Title      string `json:"title"`
			Difficulty int    `json:"difficulty"`
			PID        string `json:"pid"`
		} `json:"passed"`
	} `json:"data"`
}

type luoguUserSearchResult struct {
	UID  int64
	Name string
}

func (c *LuoguClient) searchUser(
	ctx context.Context,
	handle string,
) (luoguUserSearchResult, error) {
	response, err := fetchLuoguJSON[luoguSearchUserResponse](
		ctx,
		c.httpClient,
		c.baseURL,
		"/api/user/search",
		url.Values{"keyword": []string{handle}},
	)
	if err != nil {
		return luoguUserSearchResult{}, err
	}

	for _, user := range response.Users {
		if strings.EqualFold(user.Name, handle) {
			return luoguUserSearchResult{
				UID:  user.UID,
				Name: user.Name,
			}, nil
		}
	}

	return luoguUserSearchResult{}, fmt.Errorf("%w: handle %q", ErrLuoguUserNotFound, handle)
}

func (c *LuoguClient) fetchUserInfo(
	ctx context.Context,
	uid int64,
) (luoguUserInfoResponse, error) {
	return fetchLuoguJSON[luoguUserInfoResponse](
		ctx,
		c.httpClient,
		c.baseURL,
		fmt.Sprintf("/api/user/info/%d", uid),
		nil,
	)
}

func (c *LuoguClient) fetchPracticePage(
	ctx context.Context,
	uid int64,
) (luoguPracticePageResponse, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/user/%d/practice", c.baseURL, uid),
		nil,
	)
	if err != nil {
		return luoguPracticePageResponse{}, fmt.Errorf("build luogu practice request: %w", err)
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return luoguPracticePageResponse{}, fmt.Errorf("%w: request %s: %w", ErrLuoguAPI, request.URL.String(), err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return luoguPracticePageResponse{}, fmt.Errorf("%w: request %s returned %s", ErrLuoguAPI, request.URL.String(), httpResponse.Status)
	}

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return luoguPracticePageResponse{}, fmt.Errorf("%w: read %s: %w", ErrLuoguAPI, request.URL.String(), err)
	}

	scriptPayload, err := extractLuoguContextScript(body)
	if err != nil {
		return luoguPracticePageResponse{}, err
	}

	var response luoguPracticePageResponse
	if err := json.Unmarshal(scriptPayload, &response); err != nil {
		return luoguPracticePageResponse{}, fmt.Errorf("%w: decode %s: %w", ErrLuoguAPI, request.URL.String(), err)
	}

	return response, nil
}

func fetchLuoguJSON[T any](
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	query url.Values,
) (T, error) {
	var response T

	endpoint := baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return response, fmt.Errorf("build luogu request: %w", err)
	}

	httpResponse, err := client.Do(request)
	if err != nil {
		return response, fmt.Errorf("%w: request %s: %w", ErrLuoguAPI, endpoint, err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return response, fmt.Errorf("%w: request %s returned %s", ErrLuoguAPI, endpoint, httpResponse.Status)
	}

	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return response, fmt.Errorf("%w: decode %s: %w", ErrLuoguAPI, endpoint, err)
	}

	return response, nil
}

func extractLuoguContextScript(body []byte) ([]byte, error) {
	const (
		start = `<script id="lentille-context" type="application/json">`
		end   = `</script>`
	)

	content := string(body)
	startIdx := strings.Index(content, start)
	if startIdx == -1 {
		return nil, fmt.Errorf("%w: missing lentille-context script", ErrLuoguAPI)
	}
	startIdx += len(start)

	endIdx := strings.Index(content[startIdx:], end)
	if endIdx == -1 {
		return nil, fmt.Errorf("%w: unterminated lentille-context script", ErrLuoguAPI)
	}

	return []byte(content[startIdx : startIdx+endIdx]), nil
}
