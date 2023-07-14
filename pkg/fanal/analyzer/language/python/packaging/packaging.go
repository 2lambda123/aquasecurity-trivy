package packaging

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"golang.org/x/xerrors"

	dio "github.com/aquasecurity/go-dep-parser/pkg/io"
	"github.com/aquasecurity/go-dep-parser/pkg/python/packaging"
	godeptypes "github.com/aquasecurity/go-dep-parser/pkg/types"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer/language"
	"github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/licensing"
	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/aquasecurity/trivy/pkg/utils/fsutils"
)

func init() {
	analyzer.RegisterPostAnalyzer(analyzer.TypePythonPkg, newPackagingAnalyzer)
}

const version = 1

func newPackagingAnalyzer(opt analyzer.AnalyzerOptions) (analyzer.PostAnalyzer, error) {
	return &packagingAnalyzer{
		pkgParser:                        packaging.NewParser(),
		licenseClassifierConfidenceLevel: opt.LicenseScannerOption.ClassifierConfidenceLevel,
	}, nil
}

var (
	requiredFiles = []string{
		// .egg format
		// https://setuptools.readthedocs.io/en/latest/deprecated/python_eggs.html#eggs-and-their-formats
		".egg", // zip format
		"EGG-INFO/PKG-INFO",

		// .egg-info format: .egg-info can be a file or directory
		// https://setuptools.readthedocs.io/en/latest/deprecated/python_eggs.html#eggs-and-their-formats
		".egg-info",
		".egg-info/PKG-INFO",
	}
)

type packagingAnalyzer struct {
	pkgParser                        godeptypes.Parser
	licenseClassifierConfidenceLevel float64
}

// Analyze analyzes egg and wheel files.
func (a packagingAnalyzer) PostAnalyze(_ context.Context, input analyzer.PostAnalysisInput) (*analyzer.AnalysisResult, error) {

	var apps []types.Application

	required := func(path string, _ fs.DirEntry) bool {
		return strings.Contains(path, ".dist-info") || required(path)
	}

	err := fsutils.WalkDir(input.FS, ".", required, func(path string, d fs.DirEntry, r dio.ReadSeekerAt) error {

		// .egg file is zip format and PKG-INFO needs to be extracted from the zip file.
		if strings.HasSuffix(path, ".egg") {
			info, _ := d.Info()
			pkginfoInZip, err := a.analyzeEggZip(r, info.Size())
			if err != nil {
				return xerrors.Errorf("egg analysis error: %w", err)
			}

			// Egg archive may not contain required files, then we will get nil. Skip this archives
			if pkginfoInZip == nil {
				return nil
			}

			r = pkginfoInZip
		}
		app, err := a.parse(path, r)
		if err != nil {
			return xerrors.Errorf("parse error: %w", err)
		} else if app == nil {
			return nil
		}

		if err := a.fillAdditionalData(input.FS, path, app); err != nil {
			log.Logger.Warnf("Unable to collect additional info: %s", err)
		}

		apps = append(apps, *app)
		return nil
	})

	if err != nil {
		return nil, xerrors.Errorf("python package walk error: %w", err)
	}
	return &analyzer.AnalysisResult{
		Applications: apps,
	}, nil
}

func (a packagingAnalyzer) fillAdditionalData(fsys fs.FS, filePath string, app *types.Application) error {

	if len(app.Libraries) > 0 {
		var licenses []string
		for _, lic := range app.Libraries[0].Licenses {
			if !strings.HasPrefix(lic, "file://") {
				licenses = append(licenses, lic)
				continue
			}
			licenseFielPath := strings.TrimPrefix(lic, "file://")

			findings, err := classifyLicense(filePath, licenseFielPath, a.licenseClassifierConfidenceLevel, fsys)
			if err != nil {
				return err
			}
			// License found
			if len(findings) > 0 {
				foundLicenses := lo.Map(findings, func(finding types.LicenseFinding, _ int) string {
					return finding.Name
				})
				licenses = append(licenses, foundLicenses...)
			}
		}

		app.Libraries[0].Licenses = licenses
	}
	return nil
}

func classifyLicense(dir string, licPath string, classifierConfidenceLevel float64, fsys fs.FS) (types.LicenseFindings, error) {
	// Note that fs.FS is always slashed regardless of the platform,
	// and path.Join should be used rather than filepath.Join.
	f, err := fsys.Open(path.Join(filepath.Dir(dir), licPath))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, xerrors.Errorf("file open error: %w", err)
	}
	defer f.Close()

	l, err := licensing.Classify(licPath, f, classifierConfidenceLevel)
	if err != nil {
		return nil, xerrors.Errorf("license classify error: %w", err)
	}

	if l == nil {
		return nil, nil
	}

	return l.Findings, nil
}

func (a packagingAnalyzer) parse(path string, r dio.ReadSeekerAt) (*types.Application, error) {
	return language.Parse(types.PythonPkg, path, r, a.pkgParser)
}

func (a packagingAnalyzer) analyzeEggZip(r io.ReaderAt, size int64) (dio.ReadSeekerAt, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, xerrors.Errorf("zip reader error: %w", err)
	}

	finded, ok := lo.Find(zr.File, func(f *zip.File) bool {
		return required(f.Name)
	})
	if ok {
		return a.open(finded)
	}
	return nil, nil
}

func (a packagingAnalyzer) open(file *zip.File) (dio.ReadSeekerAt, error) {
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, xerrors.Errorf("file %s open error: %w", file.Name, err)
	}

	return bytes.NewReader(b), nil
}

func (a packagingAnalyzer) Required(filePath string, _ os.FileInfo) bool {
	return strings.Contains(filePath, ".dist-info") || required(filePath)
}

func required(filePath string) bool {
	return lo.SomeBy(requiredFiles, func(fileName string) bool {
		return strings.HasSuffix(filePath, fileName)
	})
}

func (a packagingAnalyzer) Type() analyzer.Type {
	return analyzer.TypePythonPkg
}

func (a packagingAnalyzer) Version() int {
	return version
}
