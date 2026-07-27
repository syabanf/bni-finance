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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/blackbox"
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
	// rec captures each page fetch; nil disables recording. Only bodies are
	// captured — the BNI VM token lives in the Authorization header.
	rec *blackbox.Recorder
}

// WithRecorder attaches a blackbox recorder.
func (c *Client) WithRecorder(rec *blackbox.Recorder) *Client {
	c.rec = rec
	return c
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

	started := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.rec.Record(blackbox.Call{
			Integration: "bni_vm", Direction: blackbox.Outbound,
			Method: http.MethodGet, URL: endpoint,
			Success: false, Duration: time.Since(started), Err: err,
		})
		return nil, false, fmt.Errorf("hubungi BNI VM: %w", err)
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	c.rec.Record(blackbox.Call{
		Integration: "bni_vm", Direction: blackbox.Outbound,
		Method: http.MethodGet, URL: endpoint,
		Response: raw, Status: resp.StatusCode, Success: resp.StatusCode < 300,
		Duration: time.Since(started),
	})

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, fmt.Errorf("BNI VM menolak token (HTTP %d) — periksa pengaturan bni_vm_token", resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("BNI VM membalas HTTP %d", resp.StatusCode)
	}
	if readErr != nil {
		return nil, false, fmt.Errorf("baca balasan BNI VM: %w", readErr)
	}

	var page memberPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, false, fmt.Errorf("baca balasan BNI VM: %w", err)
	}
	return page.Data, page.Pagination.HasMore, nil
}
