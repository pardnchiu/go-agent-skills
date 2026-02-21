package yahoofinance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	apiPath1 = "https://query1.finance.yahoo.com/v8/finance/chart"
	apiPath2 = "https://query2.finance.yahoo.com/v8/finance/chart"
)

var barIntervals = []string{
	"1m", "2m", "5m", "15m", "30m", "60m", "90m", "1h",
	"1d", "5d", "1wk", "1mo", "3mo",
}

var timeRanges = []string{
	"1d", "5d", "1mo", "3mo", "6mo", "1y", "2y", "5y", "10y", "ytd", "max",
}

type responseData struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol               string  `json:"symbol"`
				Currency             string  `json:"currency"`
				ExchangeName         string  `json:"exchangeName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
				RegularMarketTime    int64   `json:"regularMarketTime"`
				FiftyTwoWeekHigh     float64 `json:"fiftyTwoWeekHigh"`
				FiftyTwoWeekLow      float64 `json:"fiftyTwoWeekLow"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				Timezone             string  `json:"timezone"`
			} `json:"meta"`
			Timestamps []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type tickerData struct {
	Symbol               string          `json:"symbol"`
	Currency             string          `json:"currency"`
	ExchangeName         string          `json:"exchangeName"`
	RegularMarketPrice   float64         `json:"regularMarketPrice"`
	RegularMarketDayHigh float64         `json:"regularMarketDayHigh"`
	RegularMarketDayLow  float64         `json:"regularMarketDayLow"`
	RegularMarketVolume  int64           `json:"regularMarketVolume"`
	RegularMarketTime    int64           `json:"regularMarketTime"`
	FiftyTwoWeekHigh     float64         `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow      float64         `json:"fiftyTwoWeekLow"`
	ChartPreviousClose   float64         `json:"chartPreviousClose"`
	ChangePercent        float64         `json:"changePercent"`
	LastUpdated          time.Time       `json:"lastUpdated"`
	Candles              []candelBarData `json:"candles,omitempty"`
}

type candelBarData struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

func Fetch(ticker, barInterval, timeRange string) (string, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return "", fmt.Errorf("ticker is required")
	}

	if barInterval == "" {
		barInterval = "1m"
	}

	if timeRange == "" {
		timeRange = "1d"
	}

	if !slices.Contains(barIntervals, barInterval) {
		return "", fmt.Errorf("invalid interval: %s", barInterval)
	}

	if !slices.Contains(timeRanges, timeRange) {
		return "", fmt.Errorf("invalid range: %s", timeRange)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// * concurrent fetch 2 url to get first return
	data, err := concurrentFetch(ctx, ticker, barInterval, timeRange)
	if err != nil {
		return "", fmt.Errorf("failed to fetch ticker (%s): %w", ticker, err)
	}

	return data, nil
}

func concurrentFetch(ctx context.Context, ticker, barInterval, timeRange string) (string, error) {
	type result struct {
		data string
		err  error
	}

	ch := make(chan result, 2)
	for _, e := range []string{apiPath1, apiPath2} {
		go func(path string) {
			data, err := fetch(ctx, path, ticker, barInterval, timeRange)
			ch <- result{data, err}
		}(e)
	}

	var err error
	for range 2 {
		result := <-ch
		if result.err == nil {
			// * return with first response
			return result.data, nil
		}
		err = result.err
	}
	return "", err
}

func fetch(ctx context.Context, baseURL, ticker, barInterval, timeRange string) (string, error) {
	path := fmt.Sprintf("%s/%s?interval=%s&range=%s", baseURL, ticker, barInterval, timeRange)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://finance.yahoo.com")

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

	data, err := parse(body)
	if err != nil {
		return "", fmt.Errorf("failed to parse: %w", err)
	}
	return data, nil
}

func parse(raw []byte) (string, error) {
	var resp responseData
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Chart.Error != nil {
		return "", fmt.Errorf("API error [%d]: %s", resp.Chart.Error.Code, resp.Chart.Error.Description)
	}

	if len(resp.Chart.Result) == 0 {
		return "", fmt.Errorf("no result")
	}

	result := resp.Chart.Result[0]
	meta := result.Meta
	data := &tickerData{
		Symbol:               meta.Symbol,
		Currency:             meta.Currency,
		ExchangeName:         meta.ExchangeName,
		RegularMarketPrice:   meta.RegularMarketPrice,
		RegularMarketDayHigh: meta.RegularMarketDayHigh,
		RegularMarketDayLow:  meta.RegularMarketDayLow,
		RegularMarketVolume:  meta.RegularMarketVolume,
		RegularMarketTime:    meta.RegularMarketTime,
		FiftyTwoWeekHigh:     meta.FiftyTwoWeekHigh,
		FiftyTwoWeekLow:      meta.FiftyTwoWeekLow,
		ChartPreviousClose:   meta.ChartPreviousClose,
		LastUpdated:          time.Unix(meta.RegularMarketTime, 0),
	}

	if meta.ChartPreviousClose != 0 {
		data.ChangePercent = (meta.RegularMarketPrice - meta.ChartPreviousClose) / meta.ChartPreviousClose * 100
	}

	if len(result.Indicators.Quote) > 0 && len(result.Timestamps) > 0 {
		quote := result.Indicators.Quote[0]
		for i, ts := range result.Timestamps {
			if i >= len(quote.Close) || quote.Close[i] == nil {
				continue
			}
			bar := candelBarData{
				Time:  time.Unix(ts, 0),
				Close: *quote.Close[i],
			}
			if i < len(quote.Open) && quote.Open[i] != nil {
				bar.Open = *quote.Open[i]
			}
			if i < len(quote.High) && quote.High[i] != nil {
				bar.High = *quote.High[i]
			}
			if i < len(quote.Low) && quote.Low[i] != nil {
				bar.Low = *quote.Low[i]
			}
			if i < len(quote.Volume) && quote.Volume[i] != nil {
				bar.Volume = *quote.Volume[i]
			}
			data.Candles = append(data.Candles, bar)
		}
	}

	return format(data), nil
}

func format(data *tickerData) string {
	var sb strings.Builder

	sign := "+"
	if data.ChangePercent < 0 {
		sign = ""
	}

	sb.WriteString(fmt.Sprintf("Symbol:     %s (%s)\n", data.Symbol, data.ExchangeName))
	sb.WriteString(fmt.Sprintf("Price:      %.4f %s\n", data.RegularMarketPrice, data.Currency))
	sb.WriteString(fmt.Sprintf("Day High:   %.4f\n", data.RegularMarketDayHigh))
	sb.WriteString(fmt.Sprintf("Day Low:    %.4f\n", data.RegularMarketDayLow))
	sb.WriteString(fmt.Sprintf("Day Range:  %.4f\n", data.RegularMarketDayHigh-data.RegularMarketDayLow))
	sb.WriteString(fmt.Sprintf("52W High:   %.4f\n", data.FiftyTwoWeekHigh))
	sb.WriteString(fmt.Sprintf("52W Low:    %.4f\n", data.FiftyTwoWeekLow))
	sb.WriteString(fmt.Sprintf("52W Range:  %.4f\n", data.FiftyTwoWeekHigh-data.FiftyTwoWeekLow))
	sb.WriteString(fmt.Sprintf("Volume:     %d\n", data.RegularMarketVolume))
	sb.WriteString(fmt.Sprintf("Change:     %s%.2f%%\n", sign, data.ChangePercent))
	sb.WriteString(fmt.Sprintf("Prev Close: %.4f\n", data.ChartPreviousClose))
	sb.WriteString(fmt.Sprintf("Updated:    %s\n", data.LastUpdated.Format("2006-01-02 15:04:05 MST")))

	if len(data.Candles) > 0 {
		sb.WriteString(fmt.Sprintf("Candles:  %d bars\n", len(data.Candles)))
		for _, c := range data.Candles {
			sb.WriteString(fmt.Sprintf("  [%s] O:%.4f H:%.4f L:%.4f C:%.4f V:%d\n", c.Time.Format("15:04"), c.Open, c.High, c.Low, c.Close, c.Volume))
		}
	}

	return sb.String()
}
