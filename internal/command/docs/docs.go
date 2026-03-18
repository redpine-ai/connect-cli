package docs

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewDocsCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "docs [topic]",
		Short: "Show Connect API documentation",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams()
			url := "https://docs.redpine.ai"
			if len(args) > 0 {
				url += "/" + args[0]
			}
			ios.WriteJSON(output.NewSuccessEnvelope(map[string]string{
				"message": "Documentation available at: " + url,
				"url":     url,
			}))
			return nil
		},
	}
}
