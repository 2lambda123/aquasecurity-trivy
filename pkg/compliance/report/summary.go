package report

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/xerrors"

	"github.com/aquasecurity/table"
	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/compliance/spec"
	pkgReport "github.com/aquasecurity/trivy/pkg/report/table"
)

func BuildSummary(cr *ComplianceReport) *SummaryReport {
	ccma := make([]ControlCheckSummary, 0)
	for _, control := range cr.Results {
		ccm := ControlCheckSummary{ControlCheckID: control.ControlCheckID, ControlName: control.ControlName, ControlSeverity: control.ControlSeverity}
		if len(control.Results) == 0 { // this validation is mainly for vuln type
			if control.DefaultStatus == spec.PassStatus {
				ccm.TotalPass = 1
			}
			ccma = append(ccma, ccm)
			continue
		}
		for _, check := range control.Results {
			for _, cr := range check.Misconfigurations {
				if cr.CheckPass() {
					ccm.TotalPass++
					continue
				}
				ccm.TotalFail++
			}
			for _, cr := range check.Vulnerabilities {
				if cr.CheckPass() {
					ccm.TotalPass++
					continue
				}
				ccm.TotalFail++
			}
		}
		ccma = append(ccma, ccm)

	}
	return &SummaryReport{ReportID: cr.ID,
		ReportTitle:     cr.Title,
		SummaryControls: ccma}
}

type SummaryWriter struct {
	Output           io.Writer
	Severities       []string
	SeverityHeadings []string
	ColumnsHeading   []string
}

func NewSummaryWriter(output io.Writer, requiredSevs []dbTypes.Severity, columnHeading []string) SummaryWriter {
	var severities []string
	var severityHeadings []string
	severities, severityHeadings = getRequiredSeverities(requiredSevs)
	return SummaryWriter{
		Output:           output,
		Severities:       severities,
		SeverityHeadings: severityHeadings,
		ColumnsHeading:   columnHeading,
	}
}

// Write writes the results in a summarized table format
func (s SummaryWriter) Write(report *ComplianceReport) error {

	if _, err := fmt.Fprintln(s.Output); err != nil {
		return xerrors.Errorf("failed to write summary report: %w", err)
	}

	if _, err := fmt.Fprintf(s.Output, "Summary Report for compliance: %s\n", report.Title); err != nil {
		return xerrors.Errorf("failed to write summary report: %w", err)
	}
	sr := BuildSummary(report)
	t := table.New(s.Output)
	t.SetRowLines(false)
	configureHeader(s, t, s.ColumnsHeading)

	for _, summaryControl := range sr.SummaryControls {
		rowParts := make([]string, 0)
		rowParts = append(rowParts, s.generateSummary(summaryControl)...)
		t.AddRow(rowParts...)
	}

	t.Render()

	keyParts := make([]string, 0)
	for _, s := range s.Severities {
		keyParts = append(keyParts, fmt.Sprintf("%s=%s", s[:1], pkgReport.ColorizeSeverity(s, s)))
	}

	_, _ = fmt.Fprintln(s.Output, strings.Join(keyParts, " "))
	_, _ = fmt.Fprintln(s.Output)
	return nil
}

func (s SummaryWriter) generateSummary(summaryControls ControlCheckSummary) []string {
	percentage := calculatePercentage(summaryControls.TotalFail, summaryControls.TotalPass)
	return []string{summaryControls.ControlCheckID, summaryControls.ControlSeverity, summaryControls.ControlName, percentage}
}

func calculatePercentage(totalFail float32, totalPass float32) string {
	if totalPass == 0 && totalFail == 0 {
		return fmt.Sprintf("%.2f", 0.00) + "%"
	}
	relPass := totalPass / (totalFail + totalPass)
	return fmt.Sprintf("%.2f", relPass*100.0) + "%"
}

func getRequiredSeverities(requiredSevs []dbTypes.Severity) ([]string, []string) {
	requiredSevOrder := []dbTypes.Severity{dbTypes.SeverityCritical,
		dbTypes.SeverityHigh, dbTypes.SeverityMedium,
		dbTypes.SeverityLow, dbTypes.SeverityUnknown}
	var severities []string
	var severityHeadings []string
	for _, sev := range requiredSevOrder {
		for _, p := range requiredSevs {
			if p == sev {
				severities = append(severities, sev.String())
				severityHeadings = append(severityHeadings, strings.ToUpper(sev.String()[:1]))
				continue
			}
		}
	}
	return severities, severityHeadings
}

func configureHeader(s SummaryWriter, t *table.Table, columnHeading []string) {
	sevCount := len(s.Severities)
	headerRow := []string{columnHeading[0], columnHeading[1]}
	count := len(columnHeading) - len(headerRow)
	colSpan := []int{1, 1}
	headerAlignment := []table.Alignment{table.AlignLeft, table.AlignLeft}
	for i := 0; i < count; i++ {
		headerRow = append(headerRow, s.SeverityHeadings...)
		colSpan = append(colSpan, sevCount)
		headerAlignment = append(headerAlignment, table.AlignCenter)
	}
	t.SetHeaders(columnHeading...)
	t.SetAlignment(headerAlignment...)
	t.SetAutoMergeHeaders(true)
	t.SetHeaderColSpans(0, colSpan...)
}
