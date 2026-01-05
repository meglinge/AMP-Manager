package amp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// WebSearchRequest matches Amp's webSearch2 request format
type WebSearchRequest struct {
	Method string `json:"method"`
	Params struct {
		Objective         string   `json:"objective"`
		SearchQueries     []string `json:"searchQueries"`
		MaxResults        int      `json:"maxResults"`
		Thread            string   `json:"thread"`
		IsFreeTierRequest bool     `json:"isFreeTierRequest"`
	} `json:"params"`
}

// WebSearchResponse matches Amp's webSearch2 response format
type WebSearchResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		Results                 []SearchResult `json:"results"`
		Provider                string         `json:"provider"`
		ShowParallelAttribution bool           `json:"showParallelAttribution"`
	} `json:"result"`
	CreditsConsumed string `json:"creditsConsumed"`
	Error           string `json:"error,omitempty"`
}

type SearchResult struct {
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Excerpts []string `json:"excerpts"`
}

// DuckDuckGo HTML search response parsing
type duckDuckGoResult struct {
	Title   string
	URL     string
	Snippet string
}

// ExtractWebPageRequest matches Amp's extractWebPageContent request format
type ExtractWebPageRequest struct {
	Method string `json:"method"`
	Params struct {
		URL               string `json:"url"`
		Thread            string `json:"thread"`
		IsFreeTierRequest bool   `json:"isFreeTierRequest"`
	} `json:"params"`
}

// ExtractWebPageResponse matches Amp's extractWebPageContent response format
type ExtractWebPageResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		FullContent string   `json:"fullContent"`
		Excerpts    []string `json:"excerpts"`
		Provider    string   `json:"provider"`
	} `json:"result"`
}

// LocalWebSearchMiddleware intercepts webSearch2 and extractWebPageContent requests
func LocalWebSearchMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Request.URL.RawQuery

		// Handle extractWebPageContent
		if query == extractWebPageContentQuery {
			handleExtractWebPage(c)
			return
		}

		// Handle webSearch2
		if query != webSearchQuery {
			c.Next()
			return
		}

		// Read request body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Errorf("web_search: failed to read request body: %v", err)
			c.Next()
			return
		}

		var req WebSearchRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			log.Errorf("web_search: failed to parse request: %v", err)
			c.Next()
			return
		}

		log.Infof("web_search: handling locally - queries: %v, maxResults: %d", req.Params.SearchQueries, req.Params.MaxResults)

		// Perform local search
		results, err := performDuckDuckGoSearch(req.Params.SearchQueries, req.Params.MaxResults)
		if err != nil {
			log.Errorf("web_search: search failed: %v", err)
			// Fall back to upstream
			c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
			c.Next()
			return
		}

		// Build response
		resp := WebSearchResponse{
			OK:              true,
			CreditsConsumed: "0",
		}
		resp.Result.Results = results
		resp.Result.Provider = "local-duckduckgo"
		resp.Result.ShowParallelAttribution = false

		log.Infof("web_search: returning %d results locally", len(results))
		c.JSON(http.StatusOK, resp)
		c.Abort()
	}
}

// performDuckDuckGoSearch uses DuckDuckGo HTML search
func performDuckDuckGoSearch(queries []string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}

	var allResults []SearchResult
	seen := make(map[string]bool)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, query := range queries {
		if len(allResults) >= maxResults {
			break
		}

		results, err := searchDuckDuckGo(client, query)
		if err != nil {
			log.Warnf("web_search: query '%s' failed: %v", query, err)
			continue
		}

		for _, r := range results {
			if seen[r.URL] {
				continue
			}
			seen[r.URL] = true
			allResults = append(allResults, r)
			if len(allResults) >= maxResults {
				break
			}
		}
	}

	return allResults, nil
}

// searchDuckDuckGo performs a search using DuckDuckGo's HTML interface
func searchDuckDuckGo(client *http.Client, query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseDuckDuckGoHTML(string(body))
}

// parseDuckDuckGoHTML parses DuckDuckGo HTML results
func parseDuckDuckGoHTML(html string) ([]SearchResult, error) {
	var results []SearchResult

	// Find all result blocks
	// DuckDuckGo HTML format: <a class="result__a" href="...">title</a>
	// <a class="result__snippet" href="...">snippet</a>

	parts := strings.Split(html, "class=\"result__a\"")
	for i := 1; i < len(parts); i++ {
		part := parts[i]

		// Extract URL
		urlStart := strings.Index(part, "href=\"")
		if urlStart == -1 {
			continue
		}
		urlStart += 6
		urlEnd := strings.Index(part[urlStart:], "\"")
		if urlEnd == -1 {
			continue
		}
		rawURL := part[urlStart : urlStart+urlEnd]

		// DuckDuckGo uses redirect URLs, extract actual URL
		actualURL := extractActualURL(rawURL)
		if actualURL == "" {
			continue
		}

		// Extract title
		titleStart := strings.Index(part, ">")
		if titleStart == -1 {
			continue
		}
		titleStart++
		titleEnd := strings.Index(part[titleStart:], "</a>")
		if titleEnd == -1 {
			continue
		}
		title := cleanHTML(part[titleStart : titleStart+titleEnd])

		// Extract snippet
		snippet := ""
		snippetPart := part[titleStart+titleEnd:]
		snippetStart := strings.Index(snippetPart, "result__snippet")
		if snippetStart != -1 {
			snippetPart = snippetPart[snippetStart:]
			snipStart := strings.Index(snippetPart, ">")
			if snipStart != -1 {
				snipStart++
				snipEnd := strings.Index(snippetPart[snipStart:], "</a>")
				if snipEnd != -1 {
					snippet = cleanHTML(snippetPart[snipStart : snipStart+snipEnd])
				}
			}
		}

		result := SearchResult{
			Title: title,
			URL:   actualURL,
		}
		if snippet != "" {
			result.Excerpts = []string{snippet}
		}

		results = append(results, result)
	}

	return results, nil
}

// extractActualURL extracts the actual URL from DuckDuckGo redirect URL
func extractActualURL(ddgURL string) string {
	// DuckDuckGo format: //duckduckgo.com/l/?uddg=https%3A%2F%2F...
	if strings.Contains(ddgURL, "uddg=") {
		parts := strings.Split(ddgURL, "uddg=")
		if len(parts) > 1 {
			decoded, err := url.QueryUnescape(parts[1])
			if err == nil {
				// Remove any trailing parameters
				if idx := strings.Index(decoded, "&"); idx != -1 {
					decoded = decoded[:idx]
				}
				return decoded
			}
		}
	}

	// Direct URL
	if strings.HasPrefix(ddgURL, "http") {
		return ddgURL
	}

	return ""
}

// cleanHTML removes HTML tags and decodes entities
func cleanHTML(s string) string {
	// Remove HTML tags
	result := s
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	// Decode common HTML entities
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")

	return strings.TrimSpace(result)
}

// handleExtractWebPage handles extractWebPageContent requests locally
func handleExtractWebPage(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Errorf("extract_web: failed to read request body: %v", err)
		c.Next()
		return
	}

	var req ExtractWebPageRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Errorf("extract_web: failed to parse request: %v", err)
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		c.Next()
		return
	}

	targetURL := req.Params.URL
	if targetURL == "" {
		log.Errorf("extract_web: empty URL")
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		c.Next()
		return
	}

	log.Infof("extract_web: handling locally - URL: %s", targetURL)

	content, err := fetchWebPageContent(targetURL)
	if err != nil {
		log.Errorf("extract_web: fetch failed: %v", err)
		c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		c.Next()
		return
	}

	resp := ExtractWebPageResponse{
		OK: true,
	}
	resp.Result.FullContent = content
	resp.Result.Provider = "local"

	log.Infof("extract_web: returning %d bytes locally", len(content))
	c.JSON(http.StatusOK, resp)
	c.Abort()
}

// fetchWebPageContent fetches and extracts text content from a URL
func fetchWebPageContent(targetURL string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Return raw HTML content (matching Amp's behavior)
	return string(body), nil
}
