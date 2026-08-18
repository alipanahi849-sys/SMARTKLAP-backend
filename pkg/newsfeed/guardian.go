package newsfeed

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
)

const defaultGuardianBase = "https://content.guardianapis.com"

type guardianClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewGuardian builds The Guardian Open Platform client used for club news.
// An empty key disables it.
func NewGuardian(apiKey, baseURL string) Provider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultGuardianBase
	}
	return &guardianClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
	}
}

func (c *guardianClient) Name() string { return ProviderGuardian }

func (c *guardianClient) Enabled() bool { return c.apiKey != "" }

type guardianSearchEnvelope struct {
	Response struct {
		Status      string           `json:"status"`
		Message     string           `json:"message"`
		CurrentPage int              `json:"currentPage"`
		Pages       int              `json:"pages"`
		Results     []guardianResult `json:"results"`
		Content     *guardianResult  `json:"content"`
	} `json:"response"`
}

type guardianResult struct {
	ID                 string         `json:"id"`
	WebTitle           string         `json:"webTitle"`
	WebPublicationDate string         `json:"webPublicationDate"`
	Fields             guardianFields `json:"fields"`
}

type guardianFields struct {
	Headline  string `json:"headline"`
	TrailText string `json:"trailText"`
	Body      string `json:"body"`
	Thumbnail string `json:"thumbnail"`
}

func (c *guardianClient) Search(ctx context.Context, clubName string, page, pageSize int) (*SearchResult, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	query := searchQuery(clubName)
	if query == "" {
		return &SearchResult{Page: 1, TotalPages: 0, Items: []Article{}}, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	values := url.Values{}
	values.Set("section", "football")
	values.Set("order-by", "newest")
	values.Set("show-fields", "headline,trailText,thumbnail,body")
	values.Set("page", strconv.Itoa(page))
	values.Set("page-size", strconv.Itoa(pageSize))
	if tag, err := c.resolveFootballTag(ctx, clubName); err != nil {
		logger.Warn().Err(err).Str("club", clubName).Msg("guardian football tag lookup failed")
		values.Set("q", query)
	} else if tag != "" {
		values.Set("tag", tag)
	} else {
		values.Set("q", query)
	}

	var envelope guardianSearchEnvelope
	if err := c.get(ctx, "/search", values, &envelope); err != nil {
		return nil, err
	}
	if msg := envelope.Response.Message; envelope.Response.Status == "error" && msg != "" {
		return nil, errors.NewServiceUnavailable("News provider returned an error", fmt.Errorf("%s", msg))
	}

	items := make([]Article, 0, len(envelope.Response.Results))
	for _, row := range envelope.Response.Results {
		if article, ok := mapGuardianArticle(row); ok {
			items = append(items, article)
		}
	}
	pages := envelope.Response.Pages
	current := envelope.Response.CurrentPage
	if current < 1 {
		current = page
	}
	return &SearchResult{Items: items, Page: current, TotalPages: pages}, nil
}

func (c *guardianClient) Get(ctx context.Context, providerID string) (*Article, error) {
	if err := c.requireEnabled(); err != nil {
		return nil, err
	}
	providerID = strings.Trim(strings.TrimSpace(providerID), "/")
	if providerID == "" || strings.Contains(providerID, "..") || strings.Contains(providerID, "://") {
		return nil, errors.NewBadRequest("Invalid news ID", nil)
	}

	values := url.Values{}
	values.Set("show-fields", "headline,trailText,thumbnail,body")

	var envelope guardianSearchEnvelope
	if err := c.get(ctx, "/"+providerID, values, &envelope); err != nil {
		return nil, err
	}
	if envelope.Response.Content == nil {
		return nil, errors.NewNotFound("News article not found", nil)
	}
	article, ok := mapGuardianArticle(*envelope.Response.Content)
	if !ok {
		return nil, errors.NewNotFound("News article not found", nil)
	}
	return &article, nil
}

func (c *guardianClient) requireEnabled() error {
	if !c.Enabled() {
		return errors.NewServiceUnavailable("News provider is not configured", nil)
	}
	return nil
}

func (c *guardianClient) get(ctx context.Context, path string, query url.Values, dest any) error {
	if query == nil {
		query = url.Values{}
	}
	query.Set("api-key", c.apiKey)
	query.Set("format", "json")

	reqURL := c.baseURL + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return errors.NewInternal("Failed to build news provider request", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewServiceUnavailable("News provider is unreachable", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return errors.NewInternal("Failed to read news provider response", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return errors.NewTooManyRequests("News provider rate limit reached", nil)
	}
	if resp.StatusCode == http.StatusNotFound {
		return errors.NewNotFound("News article not found", nil)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.NewServiceUnavailable("News provider rejected the API key", nil)
	}
	if resp.StatusCode >= 400 {
		return errors.NewServiceUnavailable(fmt.Sprintf("News provider returned %d", resp.StatusCode), nil)
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		logger.Warn().Err(err).Str("path", path).Msg("news provider decode failed")
		return errors.NewInternal("Failed to decode news provider response", err)
	}
	return nil
}

func mapGuardianArticle(row guardianResult) (Article, bool) {
	id := strings.TrimSpace(row.ID)
	if id == "" {
		return Article{}, false
	}
	title := strings.TrimSpace(row.Fields.Headline)
	if title == "" {
		title = strings.TrimSpace(row.WebTitle)
	}
	if title == "" {
		return Article{}, false
	}
	body := strings.TrimSpace(row.Fields.Body)
	if body == "" {
		trail := strings.TrimSpace(html.UnescapeString(row.Fields.TrailText))
		if trail != "" {
			body = "<p>" + trail + "</p>"
		}
	}
	publishedAt, err := time.Parse(time.RFC3339, row.WebPublicationDate)
	if err != nil {
		publishedAt = time.Time{}
	}
	return Article{
		ProviderID:  id,
		Title:       title,
		BodyHTML:    body,
		ImageURL:    strings.TrimSpace(row.Fields.Thumbnail),
		PublishedAt: publishedAt,
	}, true
}

type guardianTagsEnvelope struct {
	Response struct {
		Status  string        `json:"status"`
		Message string        `json:"message"`
		Results []guardianTag `json:"results"`
	} `json:"response"`
}

type guardianTag struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	WebTitle  string `json:"webTitle"`
	SectionID string `json:"sectionId"`
}

func (c *guardianClient) resolveFootballTag(ctx context.Context, clubName string) (string, error) {
	name := strings.TrimSpace(clubName)
	if name == "" {
		return "", nil
	}
	values := url.Values{}
	values.Set("q", name)
	values.Set("section", "football")
	values.Set("page-size", "50")
	var envelope guardianTagsEnvelope
	if err := c.get(ctx, "/tags", values, &envelope); err != nil {
		return "", err
	}
	return pickFootballTag(name, envelope.Response.Results), nil
}

func pickFootballTag(clubName string, tags []guardianTag) string {
	want := normalizeClubName(clubName)
	if want == "" {
		return ""
	}
	wantWomen := strings.Contains(want, "women")
	slug := clubSlug(clubName)
	slugMatch := ""
	for _, tag := range tags {
		id := strings.TrimSpace(tag.ID)
		if id == "" || !strings.HasPrefix(strings.ToLower(id), "football/") {
			continue
		}
		if tag.Type != "" && !strings.EqualFold(tag.Type, "keyword") {
			continue
		}
		idLower := strings.ToLower(id)
		if !wantWomen && strings.Contains(idLower, "-women") {
			continue
		}
		if normalizeClubName(tag.WebTitle) == want {
			return id
		}
		if slug != "" && strings.EqualFold(id, "football/"+slug) && slugMatch == "" {
			slugMatch = id
		}
	}
	return slugMatch
}

func searchQuery(clubName string) string {
	name := strings.TrimSpace(clubName)
	name = strings.ReplaceAll(name, `"`, "")
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return `"` + name + `"`
}

func normalizeClubName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.NewReplacer(".", " ", "'", "", "’", "").Replace(name)
	return strings.Join(strings.Fields(name), " ")
}

func clubSlug(name string) string {
	name = normalizeClubName(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}
