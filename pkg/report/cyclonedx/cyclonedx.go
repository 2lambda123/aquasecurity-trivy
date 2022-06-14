package cyclonedx

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
	"golang.org/x/xerrors"
	"k8s.io/utils/clock"

	ftypes "github.com/aquasecurity/fanal/types"
	"github.com/aquasecurity/trivy/pkg/artifact/sbom"
	"github.com/aquasecurity/trivy/pkg/purl"
	"github.com/aquasecurity/trivy/pkg/sbom/cyclonedx"
	"github.com/aquasecurity/trivy/pkg/scanner/utils"
	"github.com/aquasecurity/trivy/pkg/types"
)

var (
	ErrInvalidBOMLink = xerrors.New("invalid bomLink format error")
)

const (
	// https://json-schema.org/understanding-json-schema/reference/string.html#dates-and-times
	timeLayout = "2006-01-02T15:04:05+00:00"
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
	var bom *cdx.BOM
	var err error
	if report.ArtifactType == sbom.ArtifactCycloneDX {
		bom, err = cw.vex(report.Results, report.ArtifactName)
		if err != nil {
			return xerrors.Errorf("failed to convert vex: %w", err)
		}
	} else {
		bom, err = cw.convertToBom(report)
		if err != nil {
			return xerrors.Errorf("failed to convert bom: %w", err)
		}
	}

	if err = cdx.NewBOMEncoder(cw.output, cw.format).Encode(bom); err != nil {
		return xerrors.Errorf("failed to encode bom: %w", err)
	}

	return nil
}

func (cw *Writer) vex(results types.Results, bomLink string) (*cdx.BOM, error) {
	vulnMap := map[string]cdx.Vulnerability{}
	for _, result := range results {
		for _, vuln := range result.Vulnerabilities {
			ref, err := vexRef(bomLink, vuln.Ref)
			if err != nil {
				return nil, err
			}
			if v, ok := vulnMap[vuln.VulnerabilityID]; ok {
				*v.Affects = append(*v.Affects, cyclonedx.Affects(ref, vuln.InstalledVersion))
			} else {
				vulnMap[vuln.VulnerabilityID] = cyclonedx.Vulnerability(vuln, ref)
			}
		}
	}
	vulns := maps.Values(vulnMap)
	sort.Slice(vulns, func(i, j int) bool {
		return vulns[i].ID > vulns[j].ID
	})

	bom := cdx.NewBOM()
	bom.Vulnerabilities = &vulns
	bom.Metadata = cw.newBOMMetadata()
	return bom, nil
}

func vexRef(bomLink string, bomRef string) (string, error) {
	if !strings.HasPrefix(bomLink, "urn:uuid:") {
		return "", xerrors.Errorf("%q: %w", bomLink, ErrInvalidBOMLink)
	}
	return fmt.Sprintf("%s/%d#%s", strings.Replace(bomLink, "uuid", "cdx", 1), cdx.BOMFileFormatJSON, bomRef), nil
}

func (cw *Writer) newBOMMetadata() *cdx.Metadata {
	return &cdx.Metadata{
		Timestamp: cw.clock.Now().UTC().Format(timeLayout),
		Tools: &[]cdx.Tool{
			{
				Vendor:  "aquasecurity",
				Name:    "trivy",
				Version: cw.version,
			},
		},
	}
}

func (cw *Writer) convertToBom(r types.Report) (*cdx.BOM, error) {
	bom := cdx.NewBOM()
	bom.SerialNumber = cw.options.newUUID().URN()
	metadataComponent, err := cw.reportToComponent(r)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse metadata component: %w", err)
	}

	bom.Metadata = cw.newBOMMetadata()
	bom.Metadata.Component = metadataComponent

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
	vulnMap := map[string]cdx.Vulnerability{}
	for _, result := range r.Results {
		var componentDependencies []cdx.Dependency
		bomRefMap := map[string]string{}
		for _, pkg := range result.Packages {
			pkgComponent, err := cw.pkgToComponent(result.Type, r.Metadata, pkg)
			if err != nil {
				return nil, nil, nil, xerrors.Errorf("failed to parse pkg: %w", err)
			}
			if _, ok := bomRefMap[pkg.Name+utils.FormatVersion(pkg)+pkg.FilePath]; !ok {
				bomRefMap[pkg.Name+utils.FormatVersion(pkg)+pkg.FilePath] = pkgComponent.BOMRef
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
			// Take a bom-ref
			ref := bomRefMap[vuln.PkgName+vuln.InstalledVersion+vuln.PkgPath]
			if v, ok := vulnMap[vuln.VulnerabilityID]; ok {
				// If a vulnerability depends on multiple packages,
				// it will be commonised into a single vulnerability.
				//   Vulnerability component (CVE-2020-26247)
				//     -> Library component (nokogiri /srv/app1/vendor/bundle/ruby/3.0.0/specifications/nokogiri-1.10.0.gemspec)
				//     -> Library component (nokogiri /srv/app2/vendor/bundle/ruby/3.0.0/specifications/nokogiri-1.10.0.gemspec)
				*v.Affects = append(*v.Affects, cyclonedx.Affects(ref, vuln.InstalledVersion))
			} else {
				vulnMap[vuln.VulnerabilityID] = cyclonedx.Vulnerability(vuln, ref)
			}
		}

		if isAggreated(result.Type) {
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

	vulns := maps.Values(vulnMap)
	sort.Slice(vulns, func(i, j int) bool {
		return vulns[i].ID > vulns[j].ID
	})

	dependencies = append(dependencies,
		cdx.Dependency{Ref: bomRef, Dependencies: &metadataDependencies},
	)
	return &components, &dependencies, &vulns, nil
}

func (cw *Writer) pkgToComponent(t string, meta types.Metadata, pkg ftypes.Package) (cdx.Component, error) {
	pu, err := purl.NewPackageURL(t, meta, pkg)
	if err != nil {
		return cdx.Component{}, xerrors.Errorf("failed to new package purl: %w", err)
	}
	properties := cyclonedx.Properties(pkg)
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
	if isAggreated(t) {
		properties := cyclonedx.AppendProperties(*component.Properties, cyclonedx.PropertyType, t)
		component.Properties = &properties
	}

	return component, nil
}

func (cw *Writer) reportToComponent(r types.Report) (*cdx.Component, error) {
	component := &cdx.Component{
		Name: r.ArtifactName,
	}

	properties := []cdx.Property{
		cyclonedx.Property(cyclonedx.PropertySchemaVersion, strconv.Itoa(r.SchemaVersion)),
	}

	if r.Metadata.Size != 0 {
		properties = cyclonedx.AppendProperties(properties, cyclonedx.PropertySize, strconv.FormatInt(r.Metadata.Size, 10))
	}

	switch r.ArtifactType {
	case ftypes.ArtifactContainerImage:
		component.Type = cdx.ComponentTypeContainer
		p, err := purl.NewPackageURL(purl.TypeOCI, r.Metadata, ftypes.Package{})
		if err != nil {
			return nil, xerrors.Errorf("failed to new package url for oci: %w", err)
		}
		properties = cyclonedx.AppendProperties(properties, cyclonedx.PropertyImageID, r.Metadata.ImageID)

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
		properties = cyclonedx.AppendProperties(properties, cyclonedx.PropertyRepoDigest, d)
	}
	for _, d := range r.Metadata.DiffIDs {
		properties = cyclonedx.AppendProperties(properties, cyclonedx.PropertyDiffID, d)
	}
	for _, t := range r.Metadata.RepoTags {
		properties = cyclonedx.AppendProperties(properties, cyclonedx.PropertyRepoTag, t)
	}

	component.Properties = &properties

	return component, nil
}

func (cw Writer) resultToComponent(r types.Result, osFound *ftypes.OS) cdx.Component {
	component := cdx.Component{
		Name: r.Target,
		Properties: &[]cdx.Property{
			cyclonedx.Property(cyclonedx.PropertyType, r.Type),
			cyclonedx.Property(cyclonedx.PropertyClass, string(r.Class)),
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

func isAggreated(t string) bool {
	return t == ftypes.NodePkg || t == ftypes.PythonPkg || t == ftypes.GemSpec || t == ftypes.Jar
}
