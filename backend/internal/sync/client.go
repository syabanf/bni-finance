// Package sync mirrors members and chapters from BNI Visitor Management into
// the local database.
//
// This used to run in the browser: the page fetched the BNI VM token out of
// app_settings, called the API through a Vite proxy to dodge CORS, and wrote
// the rows itself. Moving it here means the token never leaves the server and
// there is no proxy to keep working.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL is the BNI Visitor Management external API.
const DefaultBaseURL = "https://www.bni-vh.com/api/external/v1"

// pageSize is what the old client used; kept so the upstream sees the same
// request shape.
const pageSize = 200

// maxPages stops a broken `hasMore` from looping forever.
const maxPages = 200

// RemoteMember is one row as BNI VM returns it. Field names are snake_case
// because that is what the upstream sends — this is the only place in the
// backend that speaks it.
type RemoteMember struct {
	ID            string  `json:"id"`
	ChapterID     string  `json:"chapter_id"`
	Chapter       string  `json:"chapter"`
	Name          string  `json:"name"`
	Email         *string `json:"email"`
	Phone         *string `json:"phone"`
	Company       *string `json:"company"`
	BusinessField *string `json:"business_field"`
	Status        string  `json:"status"`
	JoinedDate    *string `json:"joined_date"`
	RenewalDate   *string `json:"renewal_date"`
}

type memberPage struct {
	Data       []RemoteMember `json:"data"`
	Pagination struct {
		HasMore bool `json:"hasMore"`
	} `json:"pagination"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		// A sync walks every page, so the timeout covers one request, not the
		// whole run — the caller's context bounds that.
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchMembers pages through the whole member list. BNI VM has no chapters
// endpoint, so chapters are derived from these rows.
func (c *Client) FetchMembers(ctx context.Context) ([]RemoteMember, error) {
	var all []RemoteMember

	for page, offset := 0, 0; page < maxPages; page, offset = page+1, offset+pageSize {
		batch, hasMore, err := c.fetchPage(ctx, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if !hasMore {
			return all, nil
		}
		// An upstream that keeps saying hasMore while returning nothing would
		// otherwise spin until maxPages.
		if len(batch) == 0 {
			return all, nil
		}
	}
	return nil, fmt.Errorf("sinkronisasi dihentikan setelah %d halaman — BNI VM terus melaporkan ada data berikutnya", maxPages)
}

func (c *Client) fetchPage(ctx context.Context, offset int) ([]RemoteMember, bool, error) {
	endpoint, err := url.JoinPath(c.baseURL, "members")
	if err != nil {
		return nil, false, fmt.Errorf("URL BNI VM tidak valid: %w", err)
	}
	endpoint += "?limit=" + strconv.Itoa(pageSize) + "&offset=" + strconv.Itoa(offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, fmt.Errorf("buat permintaan BNI VM: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("hubungi BNI VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, fmt.Errorf("BNI VM menolak token (HTTP %d) — periksa pengaturan bni_vm_token", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("BNI VM membalas HTTP %d", resp.StatusCode)
	}

	var page memberPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, false, fmt.Errorf("baca balasan BNI VM: %w", err)
	}
	return page.Data, page.Pagination.HasMore, nil
}
