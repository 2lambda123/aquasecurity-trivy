package sbom

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"github.com/aquasecurity/trivy/pkg/fanal/types"
)

type SBOM struct {
	ID           string
	OS           *types.OS
	Packages     []types.PackageInfo
	Applications []types.Application
}

type Unmarshaler interface {
	Unmarshal(io.Reader) (SBOM, error)
}

type Format string

const (
	FormatCycloneDXJSON = "cyclonedx-json"
	FormatCycloneDXXML  = "cyclonedx-xml"
	FormatSPDXJSON      = "spdx-json"
	FormatSPDXXML       = "spdx-xml"
	FormatUnknown       = "unknown"
)

func DetectFormat(r io.ReadSeeker) Format {
	defer r.Seek(0, io.SeekStart)

	type cyclonedx struct {
		// XML specific field
		XMLNS string `json:"-" xml:"xmlns,attr"`

		// JSON specific field
		BOMFormat string `json:"bomFormat" xml:"-"`
	}

	// Try CycloneDX JSON
	var cdxBom cyclonedx
	if err := json.NewDecoder(r).Decode(&cdxBom); err == nil {
		if cdxBom.BOMFormat == "CycloneDX" {
			return FormatCycloneDXJSON
		}
	}
	_, _ = r.Seek(0, io.SeekStart)

	// Try CycloneDX XML
	if err := xml.NewDecoder(r).Decode(&cdxBom); err == nil {
		if strings.HasPrefix(cdxBom.XMLNS, "http://cyclonedx.org") {
			return FormatCycloneDXXML
		}
	}

	// TODO: implement SPDX

	return FormatUnknown
}
