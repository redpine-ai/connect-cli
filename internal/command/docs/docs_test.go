package docs

import "testing"

func TestURLFor(t *testing.T) {
	cases := map[string]string{
		"":                DocsBaseURL,
		"overview":        DocsBaseURL, // retired app.redpine.ai slug
		"quickstart":      DocsBaseURL + "/quickstart",
		"auth":            DocsBaseURL + "/authentication",
		"Authentication":  DocsBaseURL + "/authentication",
		"/filtering/":     DocsBaseURL + "/filtering",
		"preview":         DocsBaseURL + "/preview-unlock",
		"sdk":             DocsBaseURL + "/sdks",
		"some-new-page":   DocsBaseURL + "/some-new-page", // unknown slugs pass through
		"getting-started": DocsBaseURL + "/quickstart",
	}
	for topic, want := range cases {
		if got := URLFor(topic); got != want {
			t.Errorf("URLFor(%q) = %q, want %q", topic, got, want)
		}
	}
}
