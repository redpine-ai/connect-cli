package docs

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewDocsCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "docs [topic]",
		Short: "Open Connect API documentation in browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams()
			url := "https://app.redpine.ai/docs/overview"
			if len(args) > 0 {
				url = "https://app.redpine.ai/docs/" + args[0]
			}

			if ios.IsTTY() {
				fmt.Fprintf(ios.ErrOut, "Opening %s\n", url)
				openBrowser(url)
			}

			return ios.WriteJSON(output.NewSuccessEnvelope(map[string]string{
				"url": url,
			}))
		},
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}
