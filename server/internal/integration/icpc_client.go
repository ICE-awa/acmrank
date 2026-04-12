package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

var ErrICPCAwardsAPI = errors.New("icpc awards api error")

const maxICPCAwardFeedBytes = 8 << 20
const icpcAwardCacheTTL = 5 * time.Minute

type ICPCAwardClient struct {
	feedURL    string
	httpClient *http.Client
	now        func() time.Time

	mu             sync.RWMutex
	cachedRecords  []ICPCAwardFeedRecord
	cacheExpiresAt time.Time
	fetchGroup     singleflight.Group
}

type ICPCAwardFeedRecord struct {
	ContestName string
	AwardName   string
	RankText    string
	AwardDate   time.Time
	Members     []string
	SourceURL   string
	Notes       string
}

type icpcAwardFeedEnvelope struct {
	Records []icpcAwardFeedRecordPayload `json:"records"`
}

type icpcAwardFeedRecordPayload struct {
	ContestName string   `json:"contest_name"`
	AwardName   string   `json:"award_name"`
	RankText    string   `json:"rank_text"`
	AwardDate   string   `json:"award_date"`
	Members     []string `json:"members"`
	SourceURL   string   `json:"source_url"`
	Notes       string   `json:"notes"`
}

func NewICPCAwardClient(feedURL string, timeout time.Duration) *ICPCAwardClient {
	return &ICPCAwardClient{
		feedURL: strings.TrimSpace(feedURL),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		now: time.Now,
	}
}

func (c *ICPCAwardClient) FetchAwards(
	ctx context.Context,
) ([]ICPCAwardFeedRecord, error) {
	if records, ok := c.loadCachedAwards(); ok {
		return cloneICPCAwardFeedRecords(records), nil
	}

	value, err, _ := c.fetchGroup.Do("awards", func() (any, error) {
		if records, ok := c.loadCachedAwards(); ok {
			return records, nil
		}

		records, err := c.fetchAwards(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}

		c.storeCachedAwards(records)
		return records, nil
	})
	if err != nil {
		return nil, err
	}

	records, ok := value.([]ICPCAwardFeedRecord)
	if !ok {
		return nil, fmt.Errorf("%w: invalid awards fetch result type", ErrICPCAwardsAPI)
	}

	return cloneICPCAwardFeedRecords(records), nil
}

func (c *ICPCAwardClient) fetchAwards(
	ctx context.Context,
) ([]ICPCAwardFeedRecord, error) {
	if c.feedURL == "" {
		return nil, fmt.Errorf("%w: awards feed url is not configured", ErrICPCAwardsAPI)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build icpc awards request: %w", err)
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: request %s: %w", ErrICPCAwardsAPI, c.feedURL, err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: request %s returned %s", ErrICPCAwardsAPI, c.feedURL, httpResponse.Status)
	}

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxICPCAwardFeedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrICPCAwardsAPI, c.feedURL, err)
	}
	if len(body) > maxICPCAwardFeedBytes {
		return nil, fmt.Errorf("%w: read %s exceeded %d bytes", ErrICPCAwardsAPI, c.feedURL, maxICPCAwardFeedBytes)
	}

	payloads, err := decodeICPCAwardFeed(body)
	if err != nil {
		return nil, err
	}

	records := make([]ICPCAwardFeedRecord, 0, len(payloads))
	for _, payload := range payloads {
		record, err := normalizeICPCAwardFeedRecord(payload, c.feedURL)
		if err != nil {
			log.Printf(
				"skip invalid icpc award feed record contest=%q award=%q: %v",
				payload.ContestName,
				payload.AwardName,
				err,
			)
			continue
		}

		records = append(records, record)
	}

	return records, nil
}

func (c *ICPCAwardClient) loadCachedAwards() ([]ICPCAwardFeedRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.cacheIsFreshLocked() {
		return nil, false
	}

	return c.cachedRecords, true
}

func (c *ICPCAwardClient) storeCachedAwards(records []ICPCAwardFeedRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cachedRecords = records
	c.cacheExpiresAt = c.now().UTC().Add(icpcAwardCacheTTL)
}

func (c *ICPCAwardClient) cacheIsFreshLocked() bool {
	return !c.cacheExpiresAt.IsZero() && c.cacheExpiresAt.After(c.now().UTC())
}

func cloneICPCAwardFeedRecords(
	records []ICPCAwardFeedRecord,
) []ICPCAwardFeedRecord {
	if records == nil {
		return nil
	}

	cloned := make([]ICPCAwardFeedRecord, len(records))
	copy(cloned, records)
	for idx, record := range records {
		if len(record.Members) == 0 {
			continue
		}

		cloned[idx].Members = append([]string(nil), record.Members...)
	}

	return cloned
}

func decodeICPCAwardFeed(body []byte) ([]icpcAwardFeedRecordPayload, error) {
	var envelope icpcAwardFeedEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Records != nil {
		return envelope.Records, nil
	}

	var records []icpcAwardFeedRecordPayload
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("%w: decode awards feed: %w", ErrICPCAwardsAPI, err)
	}

	return records, nil
}

func normalizeICPCAwardFeedRecord(
	payload icpcAwardFeedRecordPayload,
	defaultSourceURL string,
) (ICPCAwardFeedRecord, error) {
	awardDate, err := time.Parse(time.DateOnly, strings.TrimSpace(payload.AwardDate))
	if err != nil {
		return ICPCAwardFeedRecord{}, fmt.Errorf("%w: parse award_date %q: %w", ErrICPCAwardsAPI, payload.AwardDate, err)
	}

	members := make([]string, 0, len(payload.Members))
	for _, member := range payload.Members {
		trimmed := strings.TrimSpace(member)
		if trimmed == "" {
			continue
		}

		members = append(members, trimmed)
	}

	if len(members) == 0 {
		return ICPCAwardFeedRecord{}, fmt.Errorf("%w: award %q has no members", ErrICPCAwardsAPI, payload.ContestName)
	}

	contestName := strings.TrimSpace(payload.ContestName)
	if contestName == "" {
		return ICPCAwardFeedRecord{}, fmt.Errorf("%w: award record is missing contest_name", ErrICPCAwardsAPI)
	}

	awardName := strings.TrimSpace(payload.AwardName)
	if awardName == "" {
		return ICPCAwardFeedRecord{}, fmt.Errorf("%w: award %q is missing award_name", ErrICPCAwardsAPI, contestName)
	}

	sourceURL := strings.TrimSpace(payload.SourceURL)
	if sourceURL == "" {
		sourceURL = defaultSourceURL
	}

	return ICPCAwardFeedRecord{
		ContestName: contestName,
		AwardName:   awardName,
		RankText:    strings.TrimSpace(payload.RankText),
		AwardDate:   awardDate.UTC(),
		Members:     members,
		SourceURL:   sourceURL,
		Notes:       strings.TrimSpace(payload.Notes),
	}, nil
}
