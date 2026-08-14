package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var openAIImageURLDownloadClient = newPublicImageDownloadClient()

func newPublicImageDownloadClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, candidate := range ips {
			if !isPublicImageIP(candidate.IP) {
				continue
			}
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			err = dialErr
		}
		if err != nil {
			return nil, err
		}
		return nil, errors.New("image host has no public address")
	}
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			return validatePublicImageURL(req.URL)
		},
	}
}

func rewriteOpenAIImageURLsToBase64(ctx context.Context, body []byte, client *http.Client, maxBytes int64) ([]byte, error) {
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse image response: %w", err)
	}
	rawData, ok := response["data"].([]any)
	if !ok || len(rawData) == 0 {
		return nil, errors.New("image response has no data")
	}
	for index, rawItem := range rawData {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("image %d has invalid data", index)
		}
		if encoded, _ := item["b64_json"].(string); strings.TrimSpace(encoded) == "" {
			rawURL := firstImageResponseURL(item)
			if rawURL == "" {
				return nil, fmt.Errorf("image %d has no URL or base64 data", index)
			}
			data, err := downloadPublicImage(ctx, client, rawURL, maxBytes)
			if err != nil {
				return nil, fmt.Errorf("image %d download failed: %w", index, err)
			}
			item["b64_json"] = base64.StdEncoding.EncodeToString(data)
		}
		delete(item, "url")
		delete(item, "image_url")
		delete(item, "download_url")
	}
	rewritten, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode image response: %w", err)
	}
	if jsonValueContainsHTTPURL(response) {
		return nil, errors.New("image response still contains a provider URL")
	}
	return rewritten, nil
}

func firstImageResponseURL(item map[string]any) string {
	for _, key := range []string{"url", "image_url", "download_url"} {
		if value, ok := item[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func downloadPublicImage(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, errors.New("invalid image URL")
	}
	if err := validatePublicImageURL(parsed); err != nil {
		return nil, err
	}
	if client == nil {
		client = openAIImageURLDownloadClient
	}
	if maxBytes <= 0 {
		maxBytes = defaultImageMaxDownloadBytes
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, errors.New("invalid image request")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("image download request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("image download returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, errors.New("image download body failed")
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("image download is too large")
	}
	if len(data) == 0 || !strings.HasPrefix(http.DetectContentType(data), "image/") {
		return nil, errors.New("downloaded content is not an image")
	}
	return data, nil
}

func validatePublicImageURL(parsed *url.URL) error {
	if parsed == nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("image URL must be public HTTPS")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !isPublicImageIP(ip) {
		return errors.New("image URL must use a public address")
	}
	return nil
}

func isPublicImageIP(ip net.IP) bool {
	return ip != nil && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}

func jsonValueContainsHTTPURL(value any) bool {
	switch typed := value.(type) {
	case string:
		lower := strings.ToLower(typed)
		return strings.Contains(lower, "http://") || strings.Contains(lower, "https://")
	case []any:
		for _, item := range typed {
			if jsonValueContainsHTTPURL(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if jsonValueContainsHTTPURL(item) {
				return true
			}
		}
	}
	return false
}
