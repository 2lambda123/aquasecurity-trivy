package spec_test

import (
	"os"
	"testing"

	"github.com/aquasecurity/trivy/pkg/compliance/spec"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestGetScannerTypes(t *testing.T) {
	tests := []struct {
		name     string
		specPath string
		want     []types.SecurityCheck
	}{
		{name: "get config scanner type by check id prefix", specPath: "./testdata/spec.yaml", want: []types.SecurityCheck{types.SecurityCheckConfig}},
		{name: "get config and vuln scanners types by check id prefix", specPath: "./testdata/multi_scanner_spec.yaml", want: []types.SecurityCheck{types.SecurityCheckConfig, types.SecurityCheckVulnerability}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := os.ReadFile(tt.specPath)
			assert.NoError(t, err)
			got, err := spec.GetScannerTypes(string(b))
			assert.NoError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestCheckIDs(t *testing.T) {
	tests := []struct {
		name       string
		specPath   string
		wantConfig int
	}{
		{name: "get map of scannerType:checkIds array", specPath: "./testdata/spec.yaml", wantConfig: 29},
		{name: "get map of scannerType:checkIds array when dup ids", specPath: "./testdata/spec_dup_id.yaml", wantConfig: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr, err := ReadSpecFile(tt.specPath)
			assert.NoError(t, err)
			got := spec.ScannerCheckIDs(cr.Spec.Controls)
			assert.Equal(t, len(got["config"]), tt.wantConfig)
		})
	}
}

func TestValidateScanners(t *testing.T) {
	tests := []struct {
		name        string
		specPath    string
		expectError bool
	}{
		//{name: "spec with valid scanner", specPath: "./testdata/spec.yaml", expectError: false},
		{name: "spec with non valid scanner", specPath: "./testdata/bad_scanner_spec.yaml", expectError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compliance, err := ReadSpecFile(tt.specPath)
			assert.NoError(t, err)
			err = spec.ValidateScanners(compliance.Spec.Controls)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func ReadSpecFile(specFilePath string) (*spec.ComplianceSpec, error) {
	b, err := os.ReadFile(specFilePath)
	if err != nil {
		return nil, err
	}
	cr := spec.ComplianceSpec{}
	err = yaml.Unmarshal(b, &cr)
	if err != nil {
		return nil, err
	}
	return &cr, nil
}
