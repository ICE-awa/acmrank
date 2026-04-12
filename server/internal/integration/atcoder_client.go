package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
)

var (
	ErrAtCoderUserNotFound      = errors.New("atcoder user not found")
	ErrAtCoderAPI               = errors.New("atcoder api error")
	ErrAtCoderCredentialMissing = errors.New("atcoder credential missing")
)

const (
	atCoderSubmissionFetchConcurrency = 4
	atCoderSubmissionPageLimit        = 200
)

var (
	atCoderProfileRatingPattern = regexp.MustCompile(`(?s)<tr><th[^>]*>\s*Rating\s*</th><td>.*?<span[^>]*>(\d+)</span>`)
	atCoderProfileMaxPattern    = regexp.MustCompile(`(?s)<tr><th[^>]*>\s*Highest Rating\s*</th><td>.*?<span[^>]*>(\d+)</span>`)
)

type AtCoderCookieHeaderProvider func(context.Context) (string, error)

type AtCoderClient struct {
	baseURL              string
	httpClient           *http.Client
	now                  func() time.Time
	cookieHeaderProvider AtCoderCookieHeaderProvider
}

type AtCoderProfile struct {
	Handle      string
	DisplayName string
	Rating      *int
	MaxRating   *int
	ProfileURL  string
	Payload     json.RawMessage
	FetchedAt   time.Time
}

type AtCoderAcceptedSubmission struct {
	Handle       string
	ProblemKey   string
	ContestID    string
	TaskID       string
	ProblemName  string
	ProblemURL   string
	AcceptedAt   time.Time
	SubmissionID string
	SourceURL    string
	Payload      json.RawMessage
	FetchedAt    time.Time
}

type AtCoderContestHistoryEntry struct {
	ContestID      string
	ContestName    string
	Rank           *int
	OldRating      *int
	NewRating      *int
	RatingDelta    *int
	ParticipatedAt time.Time
	SourceURL      string
	Payload        json.RawMessage
	FetchedAt      time.Time
}

type atCoderHistoryResponse struct {
	IsRated           bool   `json:"IsRated"`
	Place             int    `json:"Place"`
	OldRating         int    `json:"OldRating"`
	NewRating         int    `json:"NewRating"`
	ContestScreenName string `json:"ContestScreenName"`
	ContestName       string `json:"ContestName"`
	ContestNameEn     string `json:"ContestNameEn"`
	EndTime           string `json:"EndTime"`
}

type atCoderSubmissionRow struct {
	AcceptedAt   time.Time
	TaskID       string
	ProblemName  string
	ProblemURL   string
	SubmissionID string
	SourceURL    string
}

func NewAtCoderClient(
	baseURL string,
	timeout time.Duration,
	cookieHeaderProvider AtCoderCookieHeaderProvider,
) *AtCoderClient {
	return &AtCoderClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		now:                  time.Now,
		cookieHeaderProvider: cookieHeaderProvider,
	}
}

func (c *AtCoderClient) CheckReady(ctx context.Context) error {
	cookieHeader, err := c.cookieHeader(ctx, true)
	if err != nil {
		return err
	}
	if cookieHeader == "" {
		return ErrAtCoderCredentialMissing
	}

	return nil
}

func (c *AtCoderClient) FetchProfile(
	ctx context.Context,
	handle string,
) (AtCoderProfile, error) {
	endpoint := fmt.Sprintf("%s/users/%s?lang=en", c.baseURL, url.PathEscape(handle))
	body, err := c.fetchPage(ctx, endpoint, false)
	if err != nil {
		return AtCoderProfile{}, err
	}
	if isAtCoderNotFoundPage(body) {
		return AtCoderProfile{}, fmt.Errorf("%w: handle %q", ErrAtCoderUserNotFound, handle)
	}

	payload := json.RawMessage(append([]byte(nil), body...))
	return AtCoderProfile{
		Handle:      handle,
		DisplayName: handle,
		Rating:      parseAtCoderProfileMetric(body, atCoderProfileRatingPattern),
		MaxRating:   parseAtCoderProfileMetric(body, atCoderProfileMaxPattern),
		ProfileURL:  fmt.Sprintf("%s/users/%s", c.baseURL, url.PathEscape(handle)),
		Payload:     payload,
		FetchedAt:   c.now().UTC(),
	}, nil
}

func (c *AtCoderClient) FetchContestHistory(
	ctx context.Context,
	handle string,
) ([]AtCoderContestHistoryEntry, error) {
	endpoint := fmt.Sprintf("%s/users/%s/history/json?lang=en", c.baseURL, url.PathEscape(handle))
	body, err := c.fetchPage(ctx, endpoint, false)
	if err != nil {
		return nil, err
	}
	if isAtCoderNotFoundPage(body) {
		return nil, fmt.Errorf("%w: handle %q", ErrAtCoderUserNotFound, handle)
	}

	var response []atCoderHistoryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %w", ErrAtCoderAPI, endpoint, err)
	}

	fetchedAt := c.now().UTC()
	result := make([]AtCoderContestHistoryEntry, 0, len(response))
	for _, item := range response {
		contestID, err := normalizeAtCoderContestID(item.ContestScreenName)
		if err != nil {
			return nil, fmt.Errorf("%w: normalize contest id: %w", ErrAtCoderAPI, err)
		}

		participatedAt, err := time.Parse(time.RFC3339, item.EndTime)
		if err != nil {
			return nil, fmt.Errorf("%w: parse contest end time %q: %w", ErrAtCoderAPI, item.EndTime, err)
		}

		payload, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshal atcoder contest payload: %w", err)
		}

		rank := item.Place
		entry := AtCoderContestHistoryEntry{
			ContestID:      contestID,
			ContestName:    atCoderContestName(item),
			Rank:           &rank,
			ParticipatedAt: participatedAt.UTC(),
			SourceURL:      fmt.Sprintf("%s/contests/%s", c.baseURL, contestID),
			Payload:        payload,
			FetchedAt:      fetchedAt,
		}
		if item.IsRated {
			oldRating := item.OldRating
			newRating := item.NewRating
			ratingDelta := item.NewRating - item.OldRating
			entry.OldRating = &oldRating
			entry.NewRating = &newRating
			entry.RatingDelta = &ratingDelta
		}

		result = append(result, entry)
	}

	return result, nil
}

func (c *AtCoderClient) FetchAcceptedSubmissions(
	ctx context.Context,
	handle string,
	contestIDs []string,
) ([]AtCoderAcceptedSubmission, error) {
	if _, err := c.cookieHeader(ctx, true); err != nil {
		return nil, err
	}

	uniqueContestIDs := dedupeStrings(contestIDs)
	fetchedAt := c.now().UTC()
	seen := make(map[string]struct{})
	result := make([]AtCoderAcceptedSubmission, 0)
	var mu sync.Mutex

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(atCoderSubmissionFetchConcurrency)

	for _, contestID := range uniqueContestIDs {
		contestID := contestID
		if contestID == "" {
			continue
		}

		group.Go(func() error {
			rows, err := c.fetchContestAcceptedSubmissions(groupCtx, handle, contestID)
			if err != nil {
				return err
			}

			mu.Lock()
			defer mu.Unlock()

			for _, row := range rows {
				if _, ok := seen[row.SubmissionID]; ok {
					continue
				}
				seen[row.SubmissionID] = struct{}{}

				payload, err := json.Marshal(map[string]any{
					"contest_id":    contestID,
					"submission_id": row.SubmissionID,
					"task_id":       row.TaskID,
					"problem_name":  row.ProblemName,
					"accepted_at":   row.AcceptedAt.UTC().Format(time.RFC3339),
					"source_url":    row.SourceURL,
				})
				if err != nil {
					return fmt.Errorf("marshal atcoder submission payload: %w", err)
				}

				result = append(result, AtCoderAcceptedSubmission{
					Handle:       handle,
					ProblemKey:   row.TaskID,
					ContestID:    contestID,
					TaskID:       row.TaskID,
					ProblemName:  row.ProblemName,
					ProblemURL:   row.ProblemURL,
					AcceptedAt:   row.AcceptedAt.UTC(),
					SubmissionID: row.SubmissionID,
					SourceURL:    row.SourceURL,
					Payload:      payload,
					FetchedAt:    fetchedAt,
				})
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *AtCoderClient) fetchContestAcceptedSubmissions(
	ctx context.Context,
	handle string,
	contestID string,
) ([]atCoderSubmissionRow, error) {
	result := make([]atCoderSubmissionRow, 0)
	seen := make(map[string]struct{})

	for page := 1; page <= atCoderSubmissionPageLimit; page++ {
		endpoint := fmt.Sprintf(
			"%s/contests/%s/submissions?f.User=%s&lang=en&page=%d",
			c.baseURL,
			url.PathEscape(contestID),
			url.QueryEscape(handle),
			page,
		)

		body, err := c.fetchPage(ctx, endpoint, true)
		if err != nil {
			return nil, err
		}

		rows, err := parseAtCoderSubmissionRows(body, c.baseURL)
		if err != nil {
			return nil, fmt.Errorf("%w: parse %s: %w", ErrAtCoderAPI, endpoint, err)
		}
		if len(rows) == 0 {
			break
		}

		added := 0
		for _, row := range rows {
			if _, ok := seen[row.SubmissionID]; ok {
				continue
			}
			seen[row.SubmissionID] = struct{}{}
			result = append(result, row)
			added++
		}
		if added == 0 {
			break
		}
	}

	return result, nil
}

func (c *AtCoderClient) fetchPage(
	ctx context.Context,
	endpoint string,
	requireCookie bool,
) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build atcoder request: %w", err)
	}

	cookieHeader, err := c.cookieHeader(ctx, requireCookie)
	if err != nil {
		return nil, err
	}
	if cookieHeader != "" {
		request.Header.Set("Cookie", cookieHeader)
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: request %s: %w", ErrAtCoderAPI, endpoint, err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrAtCoderAPI, endpoint, err)
	}

	switch httpResponse.StatusCode {
	case http.StatusOK:
	case http.StatusFound, http.StatusSeeOther:
		location := httpResponse.Header.Get("Location")
		if strings.Contains(location, "/login") && requireCookie {
			return nil, ErrAtCoderCredentialMissing
		}
		return nil, fmt.Errorf("%w: request %s returned redirect %s", ErrAtCoderAPI, endpoint, location)
	case http.StatusNotFound:
		return body, nil
	default:
		return nil, fmt.Errorf("%w: request %s returned %s", ErrAtCoderAPI, endpoint, httpResponse.Status)
	}

	if requireCookie && isAtCoderLoginPage(body) {
		return nil, ErrAtCoderCredentialMissing
	}

	return body, nil
}

func (c *AtCoderClient) cookieHeader(
	ctx context.Context,
	required bool,
) (string, error) {
	if c.cookieHeaderProvider == nil {
		if required {
			return "", ErrAtCoderCredentialMissing
		}

		return "", nil
	}

	cookieHeader, err := c.cookieHeaderProvider(ctx)
	if err != nil {
		if required {
			return "", fmt.Errorf("%w: %w", ErrAtCoderCredentialMissing, err)
		}

		return "", err
	}
	cookieHeader = strings.TrimSpace(cookieHeader)
	if cookieHeader == "" && required {
		return "", ErrAtCoderCredentialMissing
	}

	return cookieHeader, nil
}

func parseAtCoderProfileMetric(body []byte, pattern *regexp.Regexp) *int {
	matches := pattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil
	}

	value, err := strconv.Atoi(string(matches[1]))
	if err != nil {
		return nil
	}

	return &value
}

func normalizeAtCoderContestID(screenName string) (string, error) {
	trimmed := strings.TrimSpace(screenName)
	if trimmed == "" {
		return "", errors.New("contest screen name is required")
	}

	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	trimmed = strings.TrimSuffix(trimmed, "/")
	if strings.Contains(trimmed, "/") {
		trimmed = path.Base(trimmed)
	}

	trimmed = strings.TrimSuffix(trimmed, ".contest.atcoder.jp")

	contestID := strings.TrimSpace(strings.SplitN(trimmed, ".", 2)[0])
	if contestID == "" {
		return "", fmt.Errorf("invalid contest screen name %q", screenName)
	}

	return contestID, nil
}

func atCoderContestName(item atCoderHistoryResponse) string {
	if strings.TrimSpace(item.ContestNameEn) != "" {
		return item.ContestNameEn
	}

	return item.ContestName
}

func isAtCoderLoginPage(body []byte) bool {
	return bytes.Contains(body, []byte("<title>Sign In - AtCoder</title>"))
}

func isAtCoderNotFoundPage(body []byte) bool {
	return bytes.Contains(body, []byte("<title>404 Not Found - AtCoder</title>")) ||
		bytes.Contains(body, []byte("404 Page Not Found"))
}

func parseAtCoderSubmissionRows(
	body []byte,
	baseURL string,
) ([]atCoderSubmissionRow, error) {
	if isAtCoderLoginPage(body) {
		return nil, ErrAtCoderCredentialMissing
	}

	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	rows := make([]atCoderSubmissionRow, 0)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			if row, ok := parseAtCoderSubmissionRow(node, baseURL); ok {
				rows = append(rows, row)
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)

	return rows, nil
}

func parseAtCoderSubmissionRow(
	row *html.Node,
	baseURL string,
) (atCoderSubmissionRow, bool) {
	cells := htmlChildren(row, "td")
	if len(cells) == 0 {
		return atCoderSubmissionRow{}, false
	}

	timeText := ""
	taskHref := ""
	taskText := ""
	statusText := ""
	submissionHref := ""

	for _, cell := range cells {
		if timeText == "" {
			if timeNode := firstDescendant(cell, func(node *html.Node) bool {
				return node.Type == html.ElementNode && node.Data == "time"
			}); timeNode != nil {
				timeText = normalizeHTMLText(nodeText(timeNode))
			}
		}

		if taskHref == "" {
			if taskNode := firstDescendant(cell, func(node *html.Node) bool {
				return node.Type == html.ElementNode &&
					node.Data == "a" &&
					strings.Contains(attr(node, "href"), "/tasks/")
			}); taskNode != nil {
				taskHref = attr(taskNode, "href")
				taskText = normalizeHTMLText(nodeText(taskNode))
			}
		}

		if submissionHref == "" {
			if submissionNode := firstDescendant(cell, func(node *html.Node) bool {
				return node.Type == html.ElementNode &&
					node.Data == "a" &&
					strings.Contains(attr(node, "href"), "/submissions/")
			}); submissionNode != nil {
				submissionHref = attr(submissionNode, "href")
			}
		}

		cellText := strings.ToUpper(normalizeHTMLText(nodeText(cell)))
		if statusText == "" && (strings.Contains(cellText, "ACCEPTED") || cellText == "AC" || strings.Contains(cellText, " AC ")) {
			statusText = cellText
		}
	}

	if timeText == "" || taskHref == "" || submissionHref == "" {
		return atCoderSubmissionRow{}, false
	}

	status := strings.ToUpper(strings.TrimSpace(statusText))
	if !strings.Contains(status, "AC") && !strings.Contains(status, "ACCEPTED") {
		return atCoderSubmissionRow{}, false
	}

	acceptedAt, err := parseAtCoderTime(timeText)
	if err != nil {
		return atCoderSubmissionRow{}, false
	}

	taskID := path.Base(taskHref)
	submissionID := path.Base(submissionHref)
	if taskID == "" || submissionID == "" {
		return atCoderSubmissionRow{}, false
	}

	problemName := taskText
	if problemName == "" {
		problemName = taskID
	}

	return atCoderSubmissionRow{
		AcceptedAt:   acceptedAt.UTC(),
		TaskID:       taskID,
		ProblemName:  problemName,
		ProblemURL:   absoluteURL(baseURL, taskHref),
		SubmissionID: submissionID,
		SourceURL:    absoluteURL(baseURL, submissionHref),
	}, true
}

func parseAtCoderTime(raw string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05-07:00",
		time.RFC3339,
	}
	trimmed := strings.TrimSpace(raw)
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported atcoder time %q", raw)
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}

func absoluteURL(baseURL string, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(href, "/")
}

func normalizeHTMLText(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

func htmlChildren(node *html.Node, tag string) []*html.Node {
	children := make([]*html.Node, 0)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			children = append(children, child)
		}
	}

	return children
}

func firstDescendant(node *html.Node, match func(*html.Node) bool) *html.Node {
	if match(node) {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := firstDescendant(child, match); found != nil {
			return found
		}
	}

	return nil
}

func nodeText(node *html.Node) string {
	if node == nil {
		return ""
	}

	if node.Type == html.TextNode {
		return node.Data
	}

	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(nodeText(child))
		builder.WriteByte(' ')
	}

	return builder.String()
}

func attr(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}

	return ""
}
