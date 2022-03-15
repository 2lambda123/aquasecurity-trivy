package cyclonedx

import (
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"k8s.io/utils/clock"

	ftypes "github.com/aquasecurity/fanal/types"
	dtypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/aquasecurity/trivy/pkg/purl"
	"github.com/aquasecurity/trivy/pkg/scanner/utils"
	"github.com/aquasecurity/trivy/pkg/types"
)

const (
	Namespace = "aquasecurity:trivy:"

	PropertySchemaVersion = "SchemaVersion"
	PropertyType          = "Type"
	PropertyClass         = "Class"

	// Image properties
	PropertySize       = "Size"
	PropertyImageID    = "ImageID"
	PropertyRepoDigest = "RepoDigest"
	PropertyDiffID     = "DiffID"
	PropertyRepoTag    = "RepoTag"

	// Package properties
	PropertySrcName         = "SrcName"
	PropertySrcVersion      = "SrcVersion"
	PropertySrcRelease      = "SrcRelease"
	PropertySrcEpoch        = "SrcEpoch"
	PropertyModularitylabel = "Modularitylabel"
	PropertyFilePath        = "FilePath"
	PropertyLayerDigest     = "LayerDigest"
	PropertyLayerDiffID     = "LayerDiffID"
)

// Writer implements types.Writer
type Writer struct {
	output  io.Writer
	version string
	*options
}

type newUUID func() uuid.UUID

type options struct {
	format  cdx.BOMFileFormat
	clock   clock.Clock
	newUUID newUUID
}

type option func(*options)

func WithFormat(format cdx.BOMFileFormat) option {
	return func(opts *options) {
		opts.format = format
	}
}

func WithClock(clock clock.Clock) option {
	return func(opts *options) {
		opts.clock = clock
	}
}

func WithNewUUID(newUUID newUUID) option {
	return func(opts *options) {
		opts.newUUID = newUUID
	}
}

func NewWriter(output io.Writer, version string, opts ...option) Writer {
	o := &options{
		format:  cdx.BOMFileFormatJSON,
		clock:   clock.RealClock{},
		newUUID: uuid.New,
	}

	for _, opt := range opts {
		opt(o)
	}

	return Writer{
		output:  output,
		version: version,
		options: o,
	}
}

// Write writes the results in CycloneDX format
func (cw Writer) Write(report types.Report) error {
	bom, err := cw.convertToBom(report, cw.version)
	if err != nil {
		return xerrors.Errorf("failed to convert bom: %w", err)
	}

	if err = cdx.NewBOMEncoder(cw.output, cw.format).Encode(bom); err != nil {
		return xerrors.Errorf("failed to encode bom: %w", err)
	}

	return nil
}

func (cw *Writer) convertToBom(r types.Report, version string) (*cdx.BOM, error) {
	bom := cdx.NewBOM()
	bom.SerialNumber = cw.options.newUUID().URN()
	metadataComponent, err := cw.reportToComponent(r)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse metadata component: %w", err)
	}

	bom.Metadata = &cdx.Metadata{
		Timestamp: cw.clock.Now().UTC().Format(time.RFC3339Nano),
		Tools: &[]cdx.Tool{
			{
				Vendor:  "aquasecurity",
				Name:    "trivy",
				Version: version,
			},
		},
		Component: metadataComponent,
	}

	bom.Components, bom.Dependencies, bom.Vulnerabilities, err = cw.parseComponents(r, bom.Metadata.Component.BOMRef)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse components: %w", err)
	}

	return bom, nil
}

func (cw *Writer) parseComponents(r types.Report, bomRef string) (*[]cdx.Component, *[]cdx.Dependency, *[]cdx.Vulnerability, error) {
	var components []cdx.Component
	var dependencies []cdx.Dependency
	var metadataDependencies []cdx.Dependency
	libraryUniqMap := map[string]struct{}{}
	vulnMap := map[string]*cdx.Vulnerability{}
	for _, result := range r.Results {
		var componentDependencies []cdx.Dependency
		purlMap := map[string]string{}
		for _, pkg := range result.Packages {
			pkgComponent, err := cw.pkgToComponent(result.Type, r.Metadata, pkg)
			if err != nil {
				return nil, nil, nil, xerrors.Errorf("failed to parse pkg: %w", err)
			}
			if _, ok := purlMap[pkg.Name+utils.FormatVersion(pkg)+pkg.FilePath]; !ok {
				purlMap[pkg.Name+utils.FormatVersion(pkg)+pkg.FilePath] = pkgComponent.BOMRef
			}

			// When multiple lock files have the same dependency with the same name and version,
			// "bom-ref" (PURL technically) of Library components may conflict.
			// In that case, only one Library component will be added and
			// some Application components will refer to the same component.
			// e.g.
			//    Application component (/app1/package-lock.json)
			//    |
			//    |    Application component (/app2/package-lock.json)
			//    |    |
			//    └----┴----> Library component (npm package, express-4.17.3)
			//
			if _, ok := libraryUniqMap[pkgComponent.BOMRef]; !ok {
				libraryUniqMap[pkgComponent.BOMRef] = struct{}{}

				// For components
				// ref. https://cyclonedx.org/use-cases/#inventory
				//
				// TODO: All packages are flattened at the moment. We should construct dependency tree.
				components = append(components, pkgComponent)
			}

			componentDependencies = append(componentDependencies, cdx.Dependency{Ref: pkgComponent.BOMRef})
		}
		for _, vuln := range result.Vulnerabilities {
			if v, ok := vulnMap[vuln.VulnerabilityID]; ok {
				// If a vulnerability depends on multiple packages,
				// it will be commonised into a single vulnerability.
				//   Vulnerability component (CVE-2020-26247)
				//     -> Library component (nokogiri /srv/app1/specifications/app1.gemspec)
				//     -> Library component (nokogiri /srv/app2/specifications/app2.gemspec)
				*v.Affects = append(*v.Affects,
					affects(purlMap[vuln.PkgName+vuln.InstalledVersion+vuln.PkgPath], vuln.InstalledVersion),
				)
			} else {
				vulnMap[vuln.VulnerabilityID] = cw.vulnerability(vuln, purlMap)
			}
		}

		if result.Type == ftypes.NodePkg || result.Type == ftypes.PythonPkg || result.Type == ftypes.GoBinary ||
			result.Type == ftypes.GemSpec || result.Type == ftypes.Jar {
			// If a package is language-specific package that isn't associated with a lock file,
			// it will be a dependency of a component under "metadata".
			// e.g.
			//   Container component (alpine:3.15) ----------------------- #1
			//     -> Library component (npm package, express-4.17.3) ---- #2
			//     -> Library component (python package, django-4.0.2) --- #2
			//     -> etc.
			// ref. https://cyclonedx.org/use-cases/#inventory

			// Dependency graph from #1 to #2
			metadataDependencies = append(metadataDependencies, componentDependencies...)
		} else {
			// If a package is OS package, it will be a dependency of "Operating System" component.
			// e.g.
			//   Container component (alpine:3.15) --------------------- #1
			//     -> Operating System Component (Alpine Linux 3.15) --- #2
			//       -> Library component (bash-4.12) ------------------ #3
			//       -> Library component (vim-8.2)   ------------------ #3
			//       -> etc.
			//
			// Else if a package is language-specific package associated with a lock file,
			// it will be a dependency of "Application" component.
			// e.g.
			//   Container component (alpine:3.15) ------------------------ #1
			//     -> Application component (/app/package-lock.json) ------ #2
			//       -> Library component (npm package, express-4.17.3) --- #3
			//       -> Library component (npm package, lodash-4.17.21) --- #3
			//       -> etc.

			resultComponent := cw.resultToComponent(result, r.Metadata.OS)
			components = append(components, resultComponent)

			// Dependency graph from #2 to #3
			dependencies = append(dependencies,
				cdx.Dependency{Ref: resultComponent.BOMRef, Dependencies: &componentDependencies},
			)

			// Dependency graph from #1 to #2
			metadataDependencies = append(metadataDependencies, cdx.Dependency{Ref: resultComponent.BOMRef})
		}
	}
	var vulns []cdx.Vulnerability
	for _, v := range vulnMap {
		vulns = append(vulns, *v)
	}
	sort.Slice(vulns, func(i, j int) bool {
		return vulns[i].ID > vulns[j].ID
	})

	dependencies = append(dependencies,
		cdx.Dependency{Ref: bomRef, Dependencies: &metadataDependencies},
	)
	return &components, &dependencies, &vulns, nil
}

func (cw *Writer) vulnerability(vuln types.DetectedVulnerability, purlMap map[string]string) *cdx.Vulnerability {
	v := cdx.Vulnerability{
		ID:          vuln.VulnerabilityID,
		Source:      source(vuln.DataSource),
		Ratings:     ratings(vuln),
		CWEs:        cwes(vuln.CweIDs),
		Description: vuln.Description,
		Advisories:  advisories(vuln.References),
	}
	if vuln.PublishedDate != nil {
		v.Published = vuln.PublishedDate.String()
	}
	if vuln.LastModifiedDate != nil {
		v.Updated = vuln.LastModifiedDate.String()
	}
	if p, ok := purlMap[vuln.PkgName+vuln.InstalledVersion+vuln.PkgPath]; ok {
		v.Affects = &[]cdx.Affects{
			affects(p, vuln.InstalledVersion),
		}
	}

	return &v
}

func affects(ps string, v string) cdx.Affects {
	return cdx.Affects{
		Ref: ps,
		Range: &[]cdx.AffectedVersions{
			{
				Version: v,
				Status:  cdx.VulnerabilityStatusAffected,
			},
		},
	}

}

func (cw *Writer) pkgToComponent(t string, meta types.Metadata, pkg ftypes.Package) (cdx.Component, error) {
	pu, err := purl.NewPackageURL(t, meta, pkg)
	if err != nil {
		return cdx.Component{}, xerrors.Errorf("failed to new package purl: %w", err)
	}
	properties := parseProperties(pkg)
	component := cdx.Component{
		Type:       cdx.ComponentTypeLibrary,
		Name:       pkg.Name,
		Version:    pu.Version,
		BOMRef:     pu.BOMRef(),
		PackageURL: pu.ToString(),
		Properties: &properties,
	}

	if pkg.License != "" {
		component.Licenses = &cdx.Licenses{
			cdx.LicenseChoice{Expression: pkg.License},
		}
	}

	return component, nil
}

func (cw *Writer) reportToComponent(r types.Report) (*cdx.Component, error) {
	component := &cdx.Component{
		Name: r.ArtifactName,
	}

	properties := []cdx.Property{
		property(PropertySchemaVersion, strconv.Itoa(r.SchemaVersion)),
	}

	if r.Metadata.Size != 0 {
		properties = appendProperties(properties, PropertySize, strconv.FormatInt(r.Metadata.Size, 10))
	}

	switch r.ArtifactType {
	case ftypes.ArtifactContainerImage:
		component.Type = cdx.ComponentTypeContainer
		p, err := purl.NewPackageURL(purl.TypeOCI, r.Metadata, ftypes.Package{})
		if err != nil {
			return nil, xerrors.Errorf("failed to new package url for oci: %w", err)
		}
		properties = appendProperties(properties, PropertyImageID, r.Metadata.ImageID)

		if p.Type == "" {
			component.BOMRef = cw.newUUID().String()
		} else {
			component.BOMRef = p.ToString()
			component.PackageURL = p.ToString()
		}
	case ftypes.ArtifactFilesystem, ftypes.ArtifactRemoteRepository:
		component.Type = cdx.ComponentTypeApplication
		component.BOMRef = cw.newUUID().String()
	}

	for _, d := range r.Metadata.RepoDigests {
		properties = appendProperties(properties, PropertyRepoDigest, d)
	}
	for _, d := range r.Metadata.DiffIDs {
		properties = appendProperties(properties, PropertyDiffID, d)
	}
	for _, t := range r.Metadata.RepoTags {
		properties = appendProperties(properties, PropertyRepoTag, t)
	}

	component.Properties = &properties

	return component, nil
}

func (cw Writer) resultToComponent(r types.Result, osFound *ftypes.OS) cdx.Component {
	component := cdx.Component{
		Name: r.Target,
		Properties: &[]cdx.Property{
			property(PropertyType, r.Type),
			property(PropertyClass, string(r.Class)),
		},
	}

	switch r.Class {
	case types.ClassOSPkg:
		// UUID needs to be generated since Operating System Component cannot generate PURL.
		// https://cyclonedx.org/use-cases/#known-vulnerabilities
		component.BOMRef = cw.newUUID().String()
		if osFound != nil {
			component.Name = osFound.Family
			component.Version = osFound.Name
		}
		component.Type = cdx.ComponentTypeOS
	case types.ClassLangPkg:
		// UUID needs to be generated since Application Component cannot generate PURL.
		// https://cyclonedx.org/use-cases/#known-vulnerabilities
		component.BOMRef = cw.newUUID().String()
		component.Type = cdx.ComponentTypeApplication
	case types.ClassConfig:
		// TODO: Config support
		component.BOMRef = cw.newUUID().String()
		component.Type = cdx.ComponentTypeFile
	}

	return component
}

func parseProperties(pkg ftypes.Package) []cdx.Property {
	props := []struct {
		name  string
		value string
	}{
		{PropertyFilePath, pkg.FilePath},
		{PropertySrcName, pkg.SrcName},
		{PropertySrcVersion, pkg.SrcVersion},
		{PropertySrcRelease, pkg.SrcRelease},
		{PropertySrcEpoch, strconv.Itoa(pkg.SrcEpoch)},
		{PropertyModularitylabel, pkg.Modularitylabel},
		{PropertyLayerDigest, pkg.Layer.Digest},
		{PropertyLayerDiffID, pkg.Layer.DiffID},
	}

	var properties []cdx.Property
	for _, prop := range props {
		properties = appendProperties(properties, prop.name, prop.value)
	}

	return properties
}

func appendProperties(properties []cdx.Property, key, value string) []cdx.Property {
	if value == "" || (key == PropertySrcEpoch && value == "0") {
		return properties
	}
	return append(properties, property(key, value))
}

func property(key, value string) cdx.Property {
	return cdx.Property{
		Name:  Namespace + key,
		Value: value,
	}
}

func advisories(refs []string) *[]cdx.Advisory {
	var advs []cdx.Advisory
	for _, ref := range refs {
		advs = append(advs, cdx.Advisory{
			URL: ref,
		})
	}
	return &advs
}

func cwes(cweIDs []string) *[]int {
	var ret []int
	if len(cweIDs) == 0 {
		return nil
	}
	for _, id := range cweIDs {
		i, err := strconv.Atoi(strings.TrimPrefix(strings.ToLower(id), "cwe-"))
		if err != nil {
			log.Logger.Debugf("cwe id parse error: %+v", err)
		}
		ret = append(ret, i)
	}

	return &ret
}

func ratings(vulnerability types.DetectedVulnerability) *[]cdx.VulnerabilityRating {
	var rates []cdx.VulnerabilityRating
	for sourceID, cvss := range vulnerability.CVSS {
		if cvss.V3Score != 0 || cvss.V3Vector != "" {
			rate := cdx.VulnerabilityRating{
				Source: &cdx.Source{
					Name: string(sourceID),
				},
				Score:    cvss.V3Score,
				Method:   cdx.ScoringMethodCVSSv3,
				Severity: calcSeverity(cvss.V3Score),
				Vector:   cvss.V3Vector,
			}
			if strings.HasPrefix(cvss.V3Vector, "CVSS:3.1") {
				rate.Method = cdx.ScoringMethodCVSSv31
			}
			rates = append(rates, rate)
		}
		if cvss.V2Score != 0 || cvss.V2Vector != "" {
			rate := cdx.VulnerabilityRating{
				Source: &cdx.Source{
					Name: string(sourceID),
				},
				Score:    cvss.V2Score,
				Method:   cdx.ScoringMethodCVSSv2,
				Severity: calcSeverity(cvss.V2Score),
				Vector:   cvss.V2Vector,
			}
			rates = append(rates, rate)
		}
	}
	if vulnerability.DataSource != nil {
		if _, ok := vulnerability.CVSS[vulnerability.DataSource.ID]; ok {
			rate := cdx.VulnerabilityRating{
				Source: &cdx.Source{
					Name: string(vulnerability.DataSource.ID),
				},
				Severity: severity(vulnerability.Severity),
			}
			rates = append(rates, rate)
		}
	}
	sort.Slice(rates, func(i, j int) bool {
		if rates[i].Source.Name != rates[i].Source.Name {
			return rates[i].Source.Name > rates[i].Source.Name
		}
		if rates[i].Method != rates[i].Method {
			return rates[i].Method > rates[i].Method
		}
		return rates[i].Score > rates[i].Score
	})
	return &rates
}

func severity(s string) cdx.Severity {
	sev, err := dtypes.NewSeverity(s)
	if err != nil {
		log.Logger.Debugf("cyclonedx severity error: %s", err.Error())
		return cdx.SeverityUnknown
	}
	switch sev {
	case dtypes.SeverityLow:
		return cdx.SeverityLow
	case dtypes.SeverityMedium:
		return cdx.SeverityMedium
	case dtypes.SeverityHigh:
		return cdx.SeverityHigh
	case dtypes.SeverityCritical:
		return cdx.SeverityCritical
	default:
		return cdx.SeverityUnknown
	}
}

func calcSeverity(score float64) cdx.Severity {
	if score == 0 {
		return cdx.SeverityInfo
	} else if score < 4.0 {
		return cdx.SeverityLow
	} else if score < 7.0 {
		return cdx.SeverityMedium
	} else if score < 9.0 {
		return cdx.SeverityHigh
	} else {
		return cdx.SeverityCritical
	}
}

func source(source *dtypes.DataSource) *cdx.Source {
	if source == nil {
		return nil
	}

	return &cdx.Source{
		Name: string(source.ID),
		URL:  source.URL,
	}
}
