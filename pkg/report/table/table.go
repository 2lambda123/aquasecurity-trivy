package table

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/samber/lo"
	"golang.org/x/xerrors"

	"github.com/aquasecurity/table"
	"github.com/aquasecurity/tml"
	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/types"
)

var (
	SeverityColor = []func(a ...any) string{
		color.New(color.FgCyan).SprintFunc(),   // UNKNOWN
		color.New(color.FgBlue).SprintFunc(),   // LOW
		color.New(color.FgYellow).SprintFunc(), // MEDIUM
		color.New(color.FgHiRed).SprintFunc(),  // HIGH
		color.New(color.FgRed).SprintFunc(),    // CRITICAL
	}
)

// Writer implements Writer and output in tabular form
type Writer struct {
	Scanners   types.Scanners
	Severities []dbTypes.Severity
	Output     io.Writer

	// Show dependency origin tree
	Tree bool

	// Show suppressed findings
	ShowSuppressed bool

	// Hide summary table
	NoSummary bool

	// For misconfigurations
	IncludeNonFailures bool
	Trace              bool

	// For licenses
	LicenseRiskThreshold int
	IgnoredLicenses      []string
}

type Renderer interface {
	Render() string
}

// Write writes the result on standard output
func (tw Writer) Write(_ context.Context, report types.Report) error {
	if !tw.isOutputToTerminal() {
		tml.DisableFormatting()
	}

	if !tw.NoSummary {
		if err := tw.renderSummary(report); err != nil {
			return xerrors.Errorf("failed to render summary: %w", err)
		}
	}

	for _, result := range report.Results {
		// Not display a table of custom resources
		if result.Class == types.ClassCustom {
			continue
		}
		tw.write(result)
	}
	return nil
}

func (tw Writer) renderSummary(report types.Report) error {
	// Fprintln has a bug
	if err := tml.Fprintf(tw.Output, "\n<underline><bold>Report Summary</bold></underline>\n\n"); err != nil {
		return err
	}

	t := newTableWriter(tw.Output, tw.isOutputToTerminal())
	t.SetAutoMerge(false)
	t.SetColumnMaxWidth(80)

	var scanners []Scanner
	for _, scanner := range tw.Scanners {
		s := NewScanner(scanner)
		if lo.IsNil(s) {
			continue
		}
		scanners = append(scanners, s)
	}

	// It should be an impossible case.
	// But it is possible when Trivy is used as a library.
	if len(scanners) == 0 {
		return xerrors.Errorf("unable to find scanners")
	}

	headers := []string{
		"Target",
		"Type",
	}
	alignments := []table.Alignment{
		table.AlignLeft,
		table.AlignCenter,
	}
	for _, scanner := range scanners {
		headers = append(headers, scanner.Header())
		alignments = append(alignments, scanner.Alignment())
	}
	t.SetHeaders(headers...)
	t.SetAlignment(alignments...)

	for _, result := range report.Results {
		resultType := string(result.Type)
		if result.Class == types.ClassSecret {
			resultType = "text"
		} else if result.Class == types.ClassLicense || result.Class == types.ClassLicenseFile {
			resultType = "-"
		}
		rows := []string{
			result.Target,
			resultType,
		}
		for _, scanner := range scanners {
			rows = append(rows, tw.colorizeCount(scanner.Count(result)))
		}
		t.AddRows(rows)
	}
	t.Render()
	return nil
}

func (tw Writer) write(result types.Result) {
	if result.IsEmpty() && result.Class != types.ClassOSPkg {
		return
	}

	var renderer Renderer
	switch {
	// vulnerability
	case result.Class == types.ClassOSPkg || result.Class == types.ClassLangPkg:
		renderer = NewVulnerabilityRenderer(result, tw.isOutputToTerminal(), tw.Tree, tw.ShowSuppressed, tw.Severities)
	// misconfiguration
	case result.Class == types.ClassConfig:
		renderer = NewMisconfigRenderer(result, tw.Severities, tw.Trace, tw.IncludeNonFailures, tw.isOutputToTerminal())
	// secret
	case result.Class == types.ClassSecret:
		renderer = NewSecretRenderer(result.Target, result.Secrets, tw.isOutputToTerminal(), tw.Severities)
	// package license
	case result.Class == types.ClassLicense:
		renderer = NewPkgLicenseRenderer(result, tw.isOutputToTerminal(), tw.Severities)
	// file license
	case result.Class == types.ClassLicenseFile:
		renderer = NewFileLicenseRenderer(result, tw.isOutputToTerminal(), tw.Severities)
	default:
		return
	}

	_, _ = fmt.Fprint(tw.Output, renderer.Render())
}

func (tw Writer) isOutputToTerminal() bool {
	return IsOutputToTerminal(tw.Output)
}

func (tw Writer) colorizeCount(count int) string {
	if count < 0 {
		return "-"
	}
	sprintf := fmt.Sprintf
	if count != 0 && tw.isOutputToTerminal() {
		sprintf = color.New(color.FgHiRed).SprintfFunc()
	}
	return sprintf("%d", count)
}

func newTableWriter(output io.Writer, isTerminal bool) *table.Table {
	tableWriter := table.New(output)
	if isTerminal { // use ansi output if we're not piping elsewhere
		tableWriter.SetHeaderStyle(table.StyleBold)
		tableWriter.SetLineStyle(table.StyleDim)
	}
	tableWriter.SetBorders(true)
	tableWriter.SetAutoMerge(true)
	tableWriter.SetRowLines(true)

	return tableWriter
}

func summarize(specifiedSeverities []dbTypes.Severity, severityCount map[string]int) (int, []string) {
	var total int
	var severities []string
	for _, sev := range specifiedSeverities {
		severities = append(severities, sev.String())
	}

	var summaries []string
	for _, severity := range dbTypes.SeverityNames {
		if !slices.Contains(severities, severity) {
			continue
		}
		count := severityCount[severity]
		r := fmt.Sprintf("%s: %d", severity, count)
		summaries = append(summaries, r)
		total += count
	}

	return total, summaries
}

func IsOutputToTerminal(output io.Writer) bool {
	if runtime.GOOS == "windows" {
		// if its windows, we don't support formatting
		return false
	}

	if output != os.Stdout {
		return false
	}
	o, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func RenderTarget(w io.Writer, target string, isTerminal bool) {
	if isTerminal {
		// nolint
		_ = tml.Fprintf(w, "\n<underline><bold>%s</bold></underline>\n\n", target)
	} else {
		_, _ = fmt.Fprintf(w, "\n%s\n", target)
		_, _ = fmt.Fprintf(w, "%s\n", strings.Repeat("=", len(target)))
	}
}

func ColorizeSeverity(value, severity string) string {
	for i, name := range dbTypes.SeverityNames {
		if severity == name {
			return SeverityColor[i](value)
		}
	}
	return color.New(color.FgBlue).SprintFunc()(severity)
}
