package client

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"github.com/aquasecurity/trivy/pkg/types"

	"github.com/google/wire"
	"golang.org/x/xerrors"

	ftypes "github.com/aquasecurity/fanal/types"
	r "github.com/aquasecurity/trivy/pkg/rpc"
	rpc "github.com/aquasecurity/trivy/rpc/scanner"
)

// SuperSet binds the dependencies for RPC client
var SuperSet = wire.NewSet(
	NewProtobufClient,
	NewScanner,
)

// RemoteURL for RPC remote host
type RemoteURL string

// NewProtobufClient is the factory method to return RPC scanner
func NewProtobufClient(remoteURL RemoteURL) (rpc.Scanner, error) {
	var err error
	httpsInsecure := false
	value, exists := os.LookupEnv("TRIVY_INSECURE")
	if exists && value != "" {
		httpsInsecure, err = strconv.ParseBool(value)
		if err != nil {
			return nil, xerrors.Errorf("unable to parse environment variable TRIVY_INSECURE: %w", err)
		}
	}

	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.TLSClientConfig.InsecureSkipVerify = httpsInsecure

	httpClient := &http.Client{
		Transport: httpTransport,
	}

	return rpc.NewScannerProtobufClient(string(remoteURL), httpClient), nil
}

// CustomHeaders for holding HTTP headers
type CustomHeaders http.Header

// Scanner implements the RPC scanner
type Scanner struct {
	customHeaders CustomHeaders
	client        rpc.Scanner
}

// NewScanner is the factory method to return RPC Scanner
func NewScanner(customHeaders CustomHeaders, s rpc.Scanner) Scanner {
	return Scanner{customHeaders: customHeaders, client: s}
}

// Scan scans the image
func (s Scanner) Scan(target, artifactKey string, blobKeys []string, options types.ScanOptions) (types.Results, *ftypes.OS, error) {
	ctx := WithCustomHeaders(context.Background(), http.Header(s.customHeaders))

	var res *rpc.ScanResponse
	err := r.Retry(func() error {
		var err error
		res, err = s.client.Scan(ctx, &rpc.ScanRequest{
			Target:     target,
			ArtifactId: artifactKey,
			BlobIds:    blobKeys,
			Options: &rpc.ScanOptions{
				VulnType:        options.VulnType,
				SecurityChecks:  options.SecurityChecks,
				ListAllPackages: options.ListAllPackages,
			},
		})
		return err
	})
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to detect vulnerabilities via RPC: %w", err)
	}

	return r.ConvertFromRPCResults(res.Results), r.ConvertFromRPCOS(res.Os), nil
}
