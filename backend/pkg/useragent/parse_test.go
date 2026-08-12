package useragent

import "testing"

// Real User-Agent strings. The ordering traps are the point of this table:
// Edge and Opera both carry "Chrome" in their UA, Chrome carries "Safari",
// Android carries "Linux", and an iPad carries "Mac OS X".
func TestParse(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want Info
	}{
		{
			name: "Chrome on macOS",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			want: Info{Browser: "Chrome 128", OS: "macOS", Device: "desktop"},
		},
		{
			name: "Edge is not reported as Chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.2739.54",
			want: Info{Browser: "Edge 128", OS: "Windows", Device: "desktop"},
		},
		{
			name: "Opera is not reported as Chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 OPR/112.0.0.0",
			want: Info{Browser: "Opera 112", OS: "Windows", Device: "desktop"},
		},
		{
			name: "Firefox on Windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
			want: Info{Browser: "Firefox 130", OS: "Windows", Device: "desktop"},
		},
		{
			name: "Safari on macOS is not reported as Chrome",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
			want: Info{Browser: "Safari 17", OS: "macOS", Device: "desktop"},
		},
		{
			name: "Chrome on an Android phone",
			ua:   "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
			want: Info{Browser: "Chrome 128", OS: "Android", Device: "mobile"},
		},
		{
			name: "Android without Mobile is a tablet",
			ua:   "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			want: Info{Browser: "Chrome 128", OS: "Android", Device: "tablet"},
		},
		{
			name: "Samsung Internet is not reported as Chrome",
			ua:   "Mozilla/5.0 (Linux; Android 14; SAMSUNG SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/26.0 Chrome/122.0.0.0 Mobile Safari/537.36",
			want: Info{Browser: "Samsung Internet 26", OS: "Android", Device: "mobile"},
		},
		{
			name: "Safari on iPhone is iOS, not macOS",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Safari 18", OS: "iOS", Device: "mobile"},
		},
		{
			name: "iPad is a tablet, not a desktop Mac",
			ua:   "Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Safari 18", OS: "iOS", Device: "tablet"},
		},
		{
			name: "Chrome on iOS reports itself as CriOS",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/126.0.6478.54 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Chrome 126", OS: "iOS", Device: "mobile"},
		},
		{
			name: "empty header yields empty labels rather than a guess",
			ua:   "",
			want: Info{},
		},
		{
			name: "unrecognised client yields empty labels rather than a guess",
			ua:   "curl/8.7.1",
			want: Info{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.ua)
			if got != tc.want {
				t.Errorf("Parse(%q)\n got: %+v\nwant: %+v", tc.ua, got, tc.want)
			}
		})
	}
}
