package ipgeo

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout  = 3 * time.Second
	defaultCacheTTL = 12 * time.Hour
)

var defaultLookupURLs = []string{
	"https://ipwho.is/{ip}?lang=zh-CN",
	"http://ip-api.com/json/{ip}?lang=zh-CN",
}

type Result struct {
	Country  string
	Region   string
	City     string
	District string
}

func (r Result) Text() string {
	parts := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for _, item := range []string{r.Country, r.Region, r.City, r.District} {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		parts = append(parts, item)
	}
	return strings.Join(parts, " ")
}

type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

var lookupCache sync.Map

func Lookup(ctx context.Context, ip string) (Result, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" || !isPublicIP(ip) || !lookupEnabled() {
		return Result{}, nil
	}

	if value, ok := lookupCache.Load(ip); ok {
		entry, ok := value.(cacheEntry)
		if ok && time.Now().Before(entry.expiresAt) {
			return entry.result, nil
		}
		lookupCache.Delete(ip)
	}

	result, err := queryIPGeo(ctx, ip)
	if err != nil {
		return Result{}, err
	}
	if result.Text() == "" {
		return Result{}, errors.New("ip geo lookup returned empty location")
	}
	lookupCache.Store(ip, cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(defaultCacheTTL),
	})
	return result, nil
}

func lookupEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("IP_GEO_LOOKUP_ENABLED")))
	return value != "0" && value != "false" && value != "off"
}

func isPublicIP(value string) bool {
	parsed := net.ParseIP(value)
	if parsed == nil {
		return false
	}
	return !parsed.IsPrivate() &&
		!parsed.IsLoopback() &&
		!parsed.IsUnspecified() &&
		!parsed.IsMulticast() &&
		!parsed.IsLinkLocalUnicast() &&
		!parsed.IsLinkLocalMulticast()
}

type ipGeoResponse struct {
	Success    *bool  `json:"success"`
	Status     string `json:"status"`
	Error      bool   `json:"error"`
	Message    string `json:"message"`
	Reason     string `json:"reason"`
	Country    string `json:"country"`
	Region     string `json:"region"`
	RegionName string `json:"regionName"`
	City       string `json:"city"`
	District   string `json:"district"`
}

func queryIPGeo(ctx context.Context, ip string) (Result, error) {
	var lastErr error
	for _, endpoint := range lookupURLs() {
		result, err := queryIPGeoEndpoint(ctx, endpoint, ip)
		if err != nil {
			lastErr = err
			continue
		}
		if result.Text() == "" {
			lastErr = errors.New("ip geo lookup returned empty location")
			continue
		}
		return result, nil
	}
	if lastErr != nil {
		return Result{}, lastErr
	}
	return Result{}, errors.New("ip geo lookup failed")
}

func lookupURLs() []string {
	if value := strings.TrimSpace(os.Getenv("IP_GEO_LOOKUP_URLS")); value != "" {
		return splitLookupURLs(value)
	}
	if value := strings.TrimSpace(os.Getenv("IP_GEO_LOOKUP_URL")); value != "" {
		return []string{value}
	}
	return defaultLookupURLs
}

func splitLookupURLs(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		if endpoint := strings.TrimSpace(part); endpoint != "" {
			results = append(results, endpoint)
		}
	}
	return results
}

func queryIPGeoEndpoint(ctx context.Context, endpoint, ip string) (Result, error) {
	endpoint = strings.ReplaceAll(endpoint, "{ip}", url.QueryEscape(ip))

	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Result{}, errors.New("ip geo lookup failed")
	}

	var body ipGeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Result{}, err
	}
	if err := body.lookupError(); err != nil {
		return Result{}, err
	}

	region := strings.TrimSpace(body.RegionName)
	if region == "" {
		region = strings.TrimSpace(body.Region)
	}

	return Result{
		Country:  strings.TrimSpace(body.Country),
		Region:   region,
		City:     strings.TrimSpace(body.City),
		District: strings.TrimSpace(body.District),
	}, nil
}

func (r ipGeoResponse) lookupError() error {
	if r.Error {
		if message := firstNonEmpty(r.Message, r.Reason); message != "" {
			return errors.New(message)
		}
		return errors.New("ip geo lookup failed")
	}
	if r.Success != nil && !*r.Success {
		if message := firstNonEmpty(r.Message, r.Reason); message != "" {
			return errors.New(message)
		}
		return errors.New("ip geo lookup failed")
	}
	if status := strings.TrimSpace(r.Status); status != "" && !strings.EqualFold(status, "success") {
		if message := firstNonEmpty(r.Message, r.Reason, status); message != "" {
			return errors.New(message)
		}
		return errors.New("ip geo lookup failed")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value := strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
