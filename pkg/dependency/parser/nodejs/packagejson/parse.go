package packagejson

import (
	"encoding/json"
	"io"

	"github.com/samber/lo"
	"golang.org/x/xerrors"

	"github.com/aquasecurity/trivy/pkg/dependency/parser/types"
	"github.com/aquasecurity/trivy/pkg/dependency/parser/utils"
)

type packageJSON struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	License              interface{}       `json:"license"`
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	Workspaces           any               `json:"workspaces"`
}

type Package struct {
	types.Library
	Dependencies         map[string]string
	OptionalDependencies map[string]string
	DevDependencies      map[string]string
	Workspaces           []string
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(r io.Reader) (Package, error) {
	var pkgJSON packageJSON
	if err := json.NewDecoder(r).Decode(&pkgJSON); err != nil {
		return Package{}, xerrors.Errorf("JSON decode error: %w", err)
	}

	var id string
	// Name and version fields are optional
	// https://docs.npmjs.com/cli/v9/configuring-npm/package-json#name
	if pkgJSON.Name != "" && pkgJSON.Version != "" {
		id = utils.PackageID(pkgJSON.Name, pkgJSON.Version)
	}

	return Package{
		Library: types.Library{
			ID:      id,
			Name:    pkgJSON.Name,
			Version: pkgJSON.Version,
			License: parseLicense(pkgJSON.License),
		},
		Dependencies:         pkgJSON.Dependencies,
		OptionalDependencies: pkgJSON.OptionalDependencies,
		DevDependencies:      pkgJSON.DevDependencies,
		Workspaces:           parseWorkspaces(pkgJSON.Workspaces),
	}, nil
}

func parseLicense(val interface{}) string {
	// the license isn't always a string, check for legacy struct if not string
	switch v := val.(type) {
	case string:
		return v
	case map[string]interface{}:
		if license, ok := v["type"]; ok {
			return license.(string)
		}
	}
	return ""
}

// parseWorkspaces returns slice of workspaces
func parseWorkspaces(val any) []string {
	// Workspaces support 2 types - https://github.com/SchemaStore/schemastore/blob/master/src/schemas/json/package.json#L777
	switch ws := val.(type) {
	// Workspace as object (map[string][]string)
	// e.g. "workspaces": {"packages": ["packages/*", "plugins/*"]},
	case map[string]interface{}:
		// Take only workspaces for `packages` - https://classic.yarnpkg.com/blog/2018/02/15/nohoist/
		if pkgsWorkspaces, ok := ws["packages"]; ok {
			return lo.Map(pkgsWorkspaces.([]interface{}), func(workspace interface{}, _ int) string {
				return workspace.(string)
			})
		}
	// Workspace as string array
	// e.g.   "workspaces": ["packages/*", "backend"],
	case []interface{}:
		return lo.Map(ws, func(workspace interface{}, _ int) string {
			return workspace.(string)
		})
	}
	return nil
}
