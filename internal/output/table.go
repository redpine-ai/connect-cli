package output

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// RenderTable renders a borderless, left-aligned, non-wrapping table (no
// outer border, no header/row separator lines, columns separated only by
// the default cell padding), matching the layout tablewriter's v0.0.5 API
// produced with SetBorder(false)/SetHeaderLine(false)/SetAutoWrapText(false).
func RenderTable(w io.Writer, headers []string, rows [][]string) {
	table := tablewriter.NewTable(w,
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.BorderNone,
			Symbols: tw.NewSymbols(tw.StyleNone),
			Settings: tw.Settings{
				Lines:      tw.LinesNone,
				Separators: tw.SeparatorsNone,
			},
		}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
	)
	table.Header(headers)
	_ = table.Bulk(rows)
	_ = table.Render()
}
