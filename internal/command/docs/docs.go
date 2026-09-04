package docs

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/spf13/cobra"
)

// DocsBaseURL is the public documentation site.
const DocsBaseURL = "https://docs.redpine.ai/docs"

// topicAliases maps the short names this command has always accepted (and
// the slugs of the retired app.redpine.ai/docs site) onto current pages.
var topicAliases = map[string]string{
	"":                "",
	"overview":        "",
	"index":           "",
	"start":           "quickstart",
	"getting-started": "quickstart",
	"auth":            "authentication",
	"keys":            "authentication",
	"api-keys":        "authentication",
	"limits":          "rate-limits",
	"preview":         "preview-unlock",
	"unlock":          "preview-unlock",
	"assisted":        "assisted-search",
	"endpoints":       "search-endpoints",
	"sdk":             "sdks",
	"cli":             "mcp",
}

// URLFor returns the documentation URL for a topic. Unknown topics are passed
// through as slugs so new pages need no CLI release.
func URLFor(topic string) string {
	slug := strings.Trim(strings.ToLower(strings.TrimSpace(topic)), "/")
	if alias, ok := topicAliases[slug]; ok {
		slug = alias
	}
	if slug == "" {
		return DocsBaseURL
	}
	return DocsBaseURL + "/" + slug
}

func NewDocsCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "docs [topic]",
		Short: "Open the Redpine Connect documentation in your browser",
		Example: `  redpine docs                 # docs.redpine.ai
  redpine docs quickstart
  redpine docs auth            # alias for authentication
  redpine docs filtering
  redpine docs sdks`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams()
			topic := ""
			if len(args) > 0 {
				topic = args[0]
			}
			url := URLFor(topic)

			if ios.IsTTY() {
				fmt.Fprintf(ios.ErrOut, "Opening %s\n", url)
				openBrowser(url)
			}

			return ios.WriteResult(map[string]string{"url": url}, f.JSONFlag, f.PrettyFlag, func(w io.Writer) {
				fmt.Fprintln(w, url)
			})
		},
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	// #nosec G204 -- fixed command names; url is passed as a separate argv entry (no shell).
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204
	case "linux":
		cmd = exec.Command("xdg-open", url) // #nosec G204
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) // #nosec G204
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}
