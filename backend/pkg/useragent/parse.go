// Package useragent turns a browser's User-Agent header into the browser, OS
// and device labels shown in the admin activity log.
//
// The parsing is deliberately narrow: it recognises the clients this app
// actually sees and returns empty labels for anything else instead of guessing.
// Callers store the raw header alongside the parsed values, so an unrecognised
// device can still be identified by hand later.
package useragent

import (
	"regexp"
	"strings"
)

// Info is the parsed result. Every field may be empty when the header is
// missing or unrecognised.
type Info struct {
	Browser string // "Chrome 128"
	OS      string // "macOS"
	Device  string // "desktop" | "mobile" | "tablet"
}

// browserRules is scanned in order and the first match wins, because these
// tokens are nested by design: Edge, Opera and Samsung Internet all include
// "Chrome/" for compatibility, Chrome includes "Safari/", and the iOS builds of
// Chrome and Firefox announce themselves as CriOS and FxiOS while still
// including "Safari/". Reordering this list silently mislabels browsers.
var browserRules = []struct {
	name    string
	token   string
	version *regexp.Regexp
}{
	{"Edge", "Edg/", regexp.MustCompile(`Edg/(\d+)`)},
	{"Samsung Internet", "SamsungBrowser/", regexp.MustCompile(`SamsungBrowser/(\d+)`)},
	{"Opera", "OPR/", regexp.MustCompile(`OPR/(\d+)`)},
	{"Chrome", "CriOS/", regexp.MustCompile(`CriOS/(\d+)`)},
	{"Firefox", "FxiOS/", regexp.MustCompile(`FxiOS/(\d+)`)},
	{"Firefox", "Firefox/", regexp.MustCompile(`Firefox/(\d+)`)},
	{"Chrome", "Chrome/", regexp.MustCompile(`Chrome/(\d+)`)},
	{"Safari", "Safari/", regexp.MustCompile(`Version/(\d+)`)},
}

// osRules is likewise ordered: an Android UA contains "Linux", and an iPad UA
// contains "Mac OS X", so the specific platforms are tested before the generic
// ones they are built on.
var osRules = []struct {
	name  string
	token string
}{
	{"Windows", "Windows NT"},
	{"Android", "Android"},
	{"iOS", "iPhone"},
	{"iOS", "iPad"},
	{"iOS", "iPod"},
	{"macOS", "Macintosh"},
	{"Linux", "Linux"},
}

func Parse(ua string) Info {
	if strings.TrimSpace(ua) == "" {
		return Info{}
	}
	os := parseOS(ua)
	return Info{
		Browser: parseBrowser(ua),
		OS:      os,
		Device:  parseDevice(ua, os),
	}
}

func parseBrowser(ua string) string {
	for _, rule := range browserRules {
		if !strings.Contains(ua, rule.token) {
			continue
		}
		// A matching token with no parseable version still identifies the
		// browser, which is more useful than reporting nothing.
		if m := rule.version.FindStringSubmatch(ua); len(m) == 2 {
			return rule.name + " " + m[1]
		}
		return rule.name
	}
	return ""
}

func parseOS(ua string) string {
	for _, rule := range osRules {
		if strings.Contains(ua, rule.token) {
			return rule.name
		}
	}
	return ""
}

// parseDevice takes the already-resolved os so an unrecognised client is not
// reported as a desktop: "desktop" is a claim about the request, not a default.
func parseDevice(ua string, os string) string {
	switch {
	case strings.Contains(ua, "iPad"), strings.Contains(ua, "Tablet"):
		return "tablet"
	case strings.Contains(ua, "Android") && !strings.Contains(ua, "Mobile"):
		// Android tablets drop the "Mobile" token; phones keep it.
		return "tablet"
	case strings.Contains(ua, "Mobile"), strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPod"):
		return "mobile"
	case os != "":
		return "desktop"
	default:
		return ""
	}
}
