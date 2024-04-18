package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/open-policy-agent/opa/bundle"
	"golang.org/x/xerrors"
	"k8s.io/utils/clock"

	"github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/aquasecurity/trivy/pkg/oci"
)

const (
	BundleVersion    = 0 // Latest released MAJOR version for trivy-checks
	BundleRepository = "ghcr.io/aquasecurity/trivy-checks"
	policyMediaType  = "application/vnd.cncf.openpolicyagent.layer.v1.tar+gzip"
	updateInterval   = 24 * time.Hour
)

type options struct {
	artifact *oci.Artifact
	clock    clock.Clock
}

// WithOCIArtifact takes an OCI artifact
func WithOCIArtifact(art *oci.Artifact) Option {
	return func(opts *options) {
		opts.artifact = art
	}
}

// WithClock takes a clock
func WithClock(c clock.Clock) Option {
	return func(opts *options) {
		opts.clock = c
	}
}

// Option is a functional option
type Option func(*options)

// Client implements policy operations
type Client struct {
	*options
	policyDir       string
	checkBundleRepo string
	quiet           bool
}

// Metadata holds default policy metadata
type Metadata struct {
	Digest       string
	DownloadedAt time.Time
}

func (m Metadata) String() string {
	return fmt.Sprintf(`Check Bundle:
  Digest: %s
  DownloadedAt: %s
`, m.Digest, m.DownloadedAt.UTC())
}

// NewClient is the factory method for policy client
func NewClient(cacheDir string, quiet bool, checkBundleRepo string, opts ...Option) (*Client, error) {
	o := &options{
		clock: clock.RealClock{},
	}

	for _, opt := range opts {
		opt(o)
	}

	if checkBundleRepo == "" {
		checkBundleRepo = fmt.Sprintf("%s:%d", BundleRepository, BundleVersion)
	}

	return &Client{
		options:         o,
		policyDir:       filepath.Join(cacheDir, "policy"),
		checkBundleRepo: checkBundleRepo,
		quiet:           quiet,
	}, nil
}

func (c *Client) populateOCIArtifact(registryOpts types.RegistryOptions) error {
	if c.artifact == nil {
		log.Debug("Loading check bundle", log.String("repository", c.policyBundleRepo))
		art, err := oci.NewArtifact(c.policyBundleRepo, c.quiet, registryOpts)
		if err != nil {
			return xerrors.Errorf("OCI artifact error: %w", err)
		}
		c.artifact = art
	}
	return nil
}

// DownloadBuiltinPolicies download default policies from GitHub Pages
func (c *Client) DownloadBuiltinPolicies(ctx context.Context, registryOpts types.RegistryOptions) error {
	if err := c.populateOCIArtifact(registryOpts); err != nil {
		return xerrors.Errorf("OPA bundle error: %w", err)
	}

	dst := c.contentDir()
	if err := c.artifact.Download(ctx, dst, oci.DownloadOption{MediaType: policyMediaType}); err != nil {
		return xerrors.Errorf("download error: %w", err)
	}

	digest, err := c.artifact.Digest(ctx)
	if err != nil {
		return xerrors.Errorf("digest error: %w", err)
	}
	log.Debug("Digest of the built-in policies", log.String("digest", digest))

	// Update metadata.json with the new digest and the current date
	if err = c.updateMetadata(digest, c.clock.Now()); err != nil {
		return xerrors.Errorf("unable to update the policy metadata: %w", err)
	}

	return nil
}

// LoadBuiltinPolicies loads default policies
func (c *Client) LoadBuiltinPolicies() ([]string, error) {
	f, err := os.Open(c.manifestPath())
	if err != nil {
		return nil, xerrors.Errorf("manifest file open error (%s): %w", c.manifestPath(), err)
	}
	defer f.Close()

	var manifest bundle.Manifest
	if err = json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, xerrors.Errorf("json decode error (%s): %w", c.manifestPath(), err)
	}

	// If the "roots" field is not included in the manifest it defaults to [""]
	// which means that ALL data and policy must come from the bundle.
	if manifest.Roots == nil || len(*manifest.Roots) == 0 {
		return []string{c.contentDir()}, nil
	}

	var policyPaths []string
	for _, root := range *manifest.Roots {
		policyPaths = append(policyPaths, filepath.Join(c.contentDir(), root))
	}

	return policyPaths, nil
}

// NeedsUpdate returns if the default policy should be updated
func (c *Client) NeedsUpdate(ctx context.Context, registryOpts types.RegistryOptions) (bool, error) {
	meta, err := c.GetMetadata()
	if err != nil {
		return true, nil
	}

	// No need to update if it's been within a day since the last update.
	if c.clock.Now().Before(meta.DownloadedAt.Add(updateInterval)) {
		return false, nil
	}

	if err = c.populateOCIArtifact(registryOpts); err != nil {
		return false, xerrors.Errorf("OPA bundle error: %w", err)
	}

	digest, err := c.artifact.Digest(ctx)
	if err != nil {
		return false, xerrors.Errorf("digest error: %w", err)
	}

	if meta.Digest != digest {
		return true, nil
	}

	// Update DownloadedAt with the current time.
	// Otherwise, if there are no updates in the remote registry,
	// the digest will be fetched every time even after this.
	if err = c.updateMetadata(meta.Digest, time.Now()); err != nil {
		return false, xerrors.Errorf("unable to update the check metadata: %w", err)
	}

	return false, nil
}

func (c *Client) contentDir() string {
	return filepath.Join(c.policyDir, "content")
}

func (c *Client) metadataPath() string {
	return filepath.Join(c.policyDir, "metadata.json")
}

func (c *Client) manifestPath() string {
	return filepath.Join(c.contentDir(), bundle.ManifestExt)
}

func (c *Client) updateMetadata(digest string, now time.Time) error {
	f, err := os.Create(c.metadataPath())
	if err != nil {
		return xerrors.Errorf("failed to open a check manifest: %w", err)
	}
	defer f.Close()

	meta := Metadata{
		Digest:       digest,
		DownloadedAt: now,
	}

	if err = json.NewEncoder(f).Encode(meta); err != nil {
		return xerrors.Errorf("json encode error: %w", err)
	}

	return nil
}

func (c *Client) GetMetadata() (*Metadata, error) {
	f, err := os.Open(c.metadataPath())
	if err != nil {
		log.Debug("Failed to open the check metadata", log.Err(err))
		return nil, err
	}
	defer f.Close()

	var meta Metadata
	if err = json.NewDecoder(f).Decode(&meta); err != nil {
		log.Warn("Check metadata decode error", log.Err(err))
		return nil, err
	}

	return &meta, nil
}

func (c *Client) Clear() error {
	log.Info("Removing check bundle...")
	if err := os.RemoveAll(c.policyDir); err != nil {
		return xerrors.Errorf("failed to remove check bundle: %w", err)
	}
	return nil
}
