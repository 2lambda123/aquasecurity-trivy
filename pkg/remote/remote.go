package remote

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	v1types "github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/hashicorp/go-multierror"
	"github.com/samber/lo"
	"golang.org/x/xerrors"

	"github.com/aquasecurity/trivy/pkg/fanal/image/registry"
	"github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/aquasecurity/trivy/pkg/version/app"
)

type Descriptor = remote.Descriptor

// Get is a wrapper of google/go-containerregistry/pkg/v1/remote.Get
// so that it can try multiple authentication methods.
func Get(ctx context.Context, ref name.Reference, option types.RegistryOptions) (*Descriptor, error) {
	tr, err := httpTransport(option)
	if err != nil {
		return nil, xerrors.Errorf("failed to create http transport: %w", err)
	}

	mirrors, err := registryMirrors(ref, option)
	if err != nil {
		return nil, xerrors.Errorf("unable to parse mirrors: %w", err)
	}

	var errs error
	// Try each mirrors/host until it succeeds
	for _, r := range append(mirrors, ref) {
		// Try each authentication method until it succeeds
		desc, err := tryGet(ctx, tr, r, option)
		if err != nil {
			var multiErr *multierror.Error
			// authorization failed for all auth options - check other repositories
			if errors.As(err, &multiErr) {
				errs = multierror.Append(errs, multiErr.Errors...)
				continue
			}
			// Other errors
			return nil, err
		}
		return desc, nil
	}

	// No authentication for mirrors/host succeeded
	return nil, errs
}

func tryGet(ctx context.Context, tr http.RoundTripper, ref name.Reference, option types.RegistryOptions) (*Descriptor, error) {
	var errs error
	for _, authOpt := range authOptions(ctx, ref, option) {
		remoteOpts := []remote.Option{
			remote.WithTransport(tr),
			authOpt,
		}

		if option.Platform.Platform != nil {
			p, err := resolvePlatform(ref, option.Platform, remoteOpts)
			if err != nil {
				return nil, xerrors.Errorf("platform error: %w", err)
			}
			// Don't pass platform when the specified image is single-arch.
			if p.Platform != nil {
				remoteOpts = append(remoteOpts, remote.WithPlatform(*p.Platform))
			}
		}

		desc, err := remote.Get(ref, remoteOpts...)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		if option.Platform.Force {
			if err = satisfyPlatform(desc, lo.FromPtr(option.Platform.Platform)); err != nil {
				return nil, err
			}
		}
		if ref.Context().RegistryStr() != ref.Context().RegistryStr() {
			log.WithPrefix("remote").Info("Mirror was used to get remote image", log.String("image", ref.String()), log.String("mirror", ref.Context().RegistryStr()))
		}
		return desc, nil
	}
	return nil, errs
}

// Image is a wrapper of google/go-containerregistry/pkg/v1/remote.Image
// so that it can try multiple authentication methods.
func Image(ctx context.Context, ref name.Reference, option types.RegistryOptions) (v1.Image, error) {
	tr, err := httpTransport(option)
	if err != nil {
		return nil, xerrors.Errorf("failed to create http transport: %w", err)
	}

	mirrors, err := registryMirrors(ref, option)
	if err != nil {
		return nil, xerrors.Errorf("unable to parse mirrors: %w", err)
	}

	var errs error
	// Try each mirrors/origin registries until it succeeds
	for _, r := range append(mirrors, ref) {
		// Try each authentication method until it succeeds
		var image v1.Image
		image, err = tryImage(ctx, tr, r, option)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue

		}
		return image, nil
	}

	// No authentication for mirrors/host succeeded
	return nil, errs
}

// tryImage checks all auth options and tries to get v1.Image.
// If none of the auth options work - function returns multierrors for each auth option.
func tryImage(ctx context.Context, tr http.RoundTripper, ref name.Reference, option types.RegistryOptions) (v1.Image, error) {
	var errs error
	for _, authOpt := range authOptions(ctx, ref, option) {
		remoteOpts := []remote.Option{
			remote.WithTransport(tr),
			authOpt,
		}
		index, err := remote.Image(ref, remoteOpts...)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		if ref.Context().RegistryStr() != ref.Context().RegistryStr() {
			log.WithPrefix("remote").Info("Mirror was used to get remote image",
				log.String("image", ref.String()), log.String("mirror", ref.Context().RegistryStr()))
		}
		return index, nil
	}
	return nil, errs
}

// Referrers is a wrapper of google/go-containerregistry/pkg/v1/remote.Referrers
// so that it can try multiple authentication methods.
func Referrers(ctx context.Context, d name.Digest, option types.RegistryOptions) (v1.ImageIndex, error) {
	tr, err := httpTransport(option)
	if err != nil {
		return nil, xerrors.Errorf("failed to create http transport: %w", err)
	}

	var errs error
	// Try each authentication method until it succeeds
	for _, authOpt := range authOptions(ctx, d, option) {
		remoteOpts := []remote.Option{
			remote.WithTransport(tr),
			authOpt,
		}
		index, err := remote.Referrers(d, remoteOpts...)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		return index, nil
	}

	// No authentication succeeded
	return nil, errs
}

func registryMirrors(hostRef name.Reference, option types.RegistryOptions) ([]name.Reference, error) {
	var mirrors []name.Reference

	reg := hostRef.Context().RegistryStr()
	if ms, ok := option.RegistryMirrors[reg]; ok {
		for _, m := range ms {
			var nameOpts []name.Option
			if option.Insecure {
				nameOpts = append(nameOpts, name.Insecure)
			}
			mirrorImageName := strings.Replace(hostRef.Name(), reg, m, 1)
			ref, err := name.ParseReference(mirrorImageName, nameOpts...)
			if err != nil {
				return nil, xerrors.Errorf("unable to parse image from mirror registry: %w", err)
			}
			mirrors = append(mirrors, ref)
		}
	}
	return mirrors, nil
}

func httpTransport(option types.RegistryOptions) (http.RoundTripper, error) {
	d := &net.Dialer{
		Timeout: 10 * time.Minute,
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = d.DialContext
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: option.Insecure}

	if len(option.ClientCert) != 0 && len(option.ClientKey) != 0 {
		cert, err := tls.X509KeyPair(option.ClientCert, option.ClientKey)
		if err != nil {
			return nil, err
		}
		tr.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	tripper := transport.NewUserAgent(tr, fmt.Sprintf("trivy/%s", app.Version()))
	return tripper, nil
}

func authOptions(ctx context.Context, ref name.Reference, option types.RegistryOptions) []remote.Option {
	var opts []remote.Option
	for _, cred := range option.Credentials {
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: cred.Username,
			Password: cred.Password,
		}))
	}

	domain := ref.Context().RegistryStr()
	token := registry.GetToken(ctx, domain, option)
	if !lo.IsEmpty(token) {
		opts = append(opts, remote.WithAuth(&token))
	}

	switch {
	case option.RegistryToken != "":
		bearer := authn.Bearer{Token: option.RegistryToken}
		return []remote.Option{remote.WithAuth(&bearer)}
	default:
		// Use the keychain anyway at the end
		opts = append(opts, remote.WithAuthFromKeychain(authn.NewMultiKeychain(authn.DefaultKeychain, github.Keychain)))
		return opts
	}
}

// resolvePlatform resolves the OS platform for a given image reference.
// If the platform has an empty OS, the function will attempt to find the first OS
// in the image's manifest list and return the platform with the detected OS.
// It ignores the specified platform if the image is not multi-arch.
func resolvePlatform(ref name.Reference, p types.Platform, options []remote.Option) (types.Platform, error) {
	if p.OS != "" {
		return p, nil
	}

	// OS wildcard, implicitly pick up the first os found in the image list.
	// e.g. */amd64
	d, err := remote.Get(ref, options...)
	if err != nil {
		return types.Platform{}, xerrors.Errorf("image get error: %w", err)
	}
	switch d.MediaType {
	case v1types.OCIManifestSchema1, v1types.DockerManifestSchema2:
		// We want an index but the registry has an image, not multi-arch. We just ignore "--platform".
		log.Debug("Ignore `--platform` as the image is not multi-arch")
		return types.Platform{}, nil
	case v1types.OCIImageIndex, v1types.DockerManifestList:
		// These are expected.
	}

	index, err := d.ImageIndex()
	if err != nil {
		return types.Platform{}, xerrors.Errorf("image index error: %w", err)
	}

	m, err := index.IndexManifest()
	if err != nil {
		return types.Platform{}, xerrors.Errorf("remote index manifest error: %w", err)
	}
	if len(m.Manifests) == 0 {
		log.Debug("Ignore '--platform' as the image is not multi-arch")
		return types.Platform{}, nil
	}
	if m.Manifests[0].Platform != nil {
		newPlatform := p.DeepCopy()
		// Replace with the detected OS
		// e.g. */amd64 => linux/amd64
		newPlatform.OS = m.Manifests[0].Platform.OS

		// Return the platform with the found OS
		return types.Platform{
			Platform: newPlatform,
			Force:    p.Force,
		}, nil
	}
	return types.Platform{}, nil
}

func satisfyPlatform(desc *remote.Descriptor, platform v1.Platform) error {
	img, err := desc.Image()
	if err != nil {
		return err
	}
	c, err := img.ConfigFile()
	if err != nil {
		return err
	}
	if !lo.FromPtr(c.Platform()).Satisfies(platform) {
		return xerrors.Errorf("the specified platform not found")
	}
	return nil
}
