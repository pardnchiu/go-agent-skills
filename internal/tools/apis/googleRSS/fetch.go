package googleRSS

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	apiPath = "https://news.google.com/rss/search"
)

var timeRanges = []string{
	"1h", "3h", "6h", "12h", "24h", "7d",
}

type responseData struct {
	Channel struct {
		Items []responseItemData `xml:"item"`
	} `xml:"channel"`
}

type responseItemData struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Source      struct {
		URL  string `xml:"url,attr"`
		Name string `xml:",chardata"`
	} `xml:"source"`
}

func Fetch(keyword, timeRange, language string) (string, error) {
	if keyword == "" {
		return "", fmt.Errorf("keyword is required")
	}

	if timeRange == "" {
		timeRange = "7d"
	}

	if language == "" {
		language = "TW:zh-Hant"
	}

	if !slices.Contains(timeRanges, timeRange) {
		return "", fmt.Errorf("invalid interval: %s", timeRange)
	}

	parts := strings.SplitN(language, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid lang format: %s", language)
	}
	geo, lang := parts[0], parts[1]

	q := fmt.Sprintf("%s when:%s", keyword, timeRange)
	requsetPath := fmt.Sprintf("%s?q=%s&hl=%s&gl=%s&ceid=%s",
		apiPath,
		url.QueryEscape(q),
		url.QueryEscape(lang),
		url.QueryEscape(geo),
		url.QueryEscape(language),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	items, err := fetch(ctx, requsetPath)
	if err != nil {
		return "", fmt.Errorf("failed to fetch: %w", err)
	}

	return items, nil
}

func fetch(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read: %w", err)
	}

	var root responseData
	if err := xml.Unmarshal(body, &root); err != nil {
		return "", fmt.Errorf("failed to parse: %w", err)
	}

	if len(root.Channel.Items) == 0 {
		return "", fmt.Errorf("no result")
	}

	// * remove duplicates
	items := deduplicate(root.Channel.Items)

	return format(items), nil
}

func deduplicate(items []responseItemData) []responseItemData {
	done := make(map[uint64]bool)
	newItems := make([]responseItemData, 0, len(items))

	for _, item := range items {
		key := hash(item.Title, item.Source.Name)

		if !done[key] {
			done[key] = true
			newItems = append(newItems, item)
		}
	}

	return newItems
}

func hash(parts ...string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	s := strings.Join(parts, "")
	hash := uint64(offset64)
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime64
	}
	return hash
}

func format(items []responseItemData) string {
	var sb strings.Builder
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))

		if item.Source.URL != "" {
			sb.WriteString(fmt.Sprintf("   來源網站: %s (%s)\n", item.Source.URL, item.Source.Name))
		} else if item.Source.Name != "" {
			sb.WriteString(fmt.Sprintf("   來源: %s\n", item.Source.Name))
		}

		if item.PubDate != "" {
			sb.WriteString(fmt.Sprintf("   發布時間: %s\n", item.PubDate))
		}

		sb.WriteString(fmt.Sprintf("   Google News: %s\n\n", item.Link))
	}

	return sb.String()
}
