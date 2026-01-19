package crawler

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func NormalizeURL(baseURL, rawURL string) (string, error) {
	if rawURL == "" {
		return "", nil
	}

	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "#" ||
		strings.HasPrefix(rawURL, "javascript:") ||
		strings.HasPrefix(rawURL, "mailto:") {
		return "", nil
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	u, err := base.Parse(rawURL)
	if err != nil {
		return "", err
	}

	u.Fragment = ""

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if u.Path == "" {
		u.Path = "/"
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", nil
	}

	return u.String(), nil
}

func FilterSameDomain(urls []string, domain string) []string {
	var out []string
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := u.Hostname()
		if host == "" {
			continue
		}
		if host == domain || strings.HasSuffix(host, "."+domain) {
			out = append(out, raw)
		}
	}
	return out
}

func ExtractDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("invalid URL (missing host): %q", rawURL)
	}

	return publicsuffix.EffectiveTLDPlusOne(host)
}

func Unique(list []string) []string {
	seen := make(map[string]struct{}, len(list))
	out := make([]string, 0, len(list))

	for _, s := range list {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
