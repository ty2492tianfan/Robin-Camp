package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

func fetchBoxOffice(title string) (*upstreamBoxOffice, int, error) {
	if strings.TrimSpace(title) == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("missing required query parameter: title")
	}

	baseURL := os.Getenv("BOXOFFICE_URL")
	apiKey := os.Getenv("BOXOFFICE_API_KEY")
	if baseURL == "" {
		return nil, 0, fmt.Errorf("BOXOFFICE_URL is not set")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, 0, err
	}
	u.Path = path.Join(u.Path, "boxoffice")

	q := u.Query()
	q.Set("title", title)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		io.Copy(io.Discard, resp.Body)
		return nil, resp.StatusCode, nil
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, resp.StatusCode, fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	var box upstreamBoxOffice
	if err := json.NewDecoder(resp.Body).Decode(&box); err != nil {
		return nil, resp.StatusCode, err
	}
	return &box, resp.StatusCode, nil
}
