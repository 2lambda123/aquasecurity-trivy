package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/aquasecurity/trivy/pkg/utils/fsutils"
)

func CopyFile(t *testing.T, src, dst string) {
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	require.NoError(t, err)

	_, err = fsutils.CopyFile(src, dst)
	require.NoError(t, err)
}

// CopyDir copies the directory content from src to dst.
// It supports only simple cases for testing.
func CopyDir(t *testing.T, src, dst string) {
	srcInfo, err := os.Stat(src)
	require.NoError(t, err)

	err = os.MkdirAll(dst, srcInfo.Mode())
	require.NoError(t, err)

	entries, err := os.ReadDir(src)
	require.NoError(t, err)

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			CopyDir(t, srcPath, dstPath)
		} else {
			_, err = fsutils.CopyFile(srcPath, dstPath)
			require.NoError(t, err)
		}
	}
}

func MustWriteYAML(t *testing.T, path string, data interface{}) {
	t.Helper()
	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0755))

	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, yaml.NewEncoder(f).Encode(data))
}

func MustReadYAML(t *testing.T, path string, out interface{}) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	require.NoError(t, yaml.NewDecoder(f).Decode(out))
}

func MustMkdirAll(t *testing.T, dir string) {
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
}

func MustReadJSON(t *testing.T, filePath string, v interface{}) {
	b, err := os.ReadFile(filePath)
	require.NoError(t, err)
	err = json.Unmarshal(b, v)
	require.NoError(t, err)
}

func MustWriteJSON(t *testing.T, filePath string, v interface{}) {
	data, err := json.Marshal(v)
	require.NoError(t, err)

	MustWriteFile(t, filePath, data)
}

func MustWriteFile(t *testing.T, filePath string, content []byte) {
	dir := filepath.Dir(filePath)
	MustMkdirAll(t, dir)

	err := os.WriteFile(filePath, content, 0744)
	require.NoError(t, err)
}
