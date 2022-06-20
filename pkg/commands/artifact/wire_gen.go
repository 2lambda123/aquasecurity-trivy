// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package artifact

import (
	"context"
	"github.com/aquasecurity/fanal/applier"
	"github.com/aquasecurity/fanal/artifact"
	image2 "github.com/aquasecurity/fanal/artifact/image"
	local2 "github.com/aquasecurity/fanal/artifact/local"
	sbom2 "github.com/aquasecurity/trivy/pkg/artifact/sbom"
	"github.com/aquasecurity/fanal/artifact/remote"
	"github.com/aquasecurity/fanal/cache"
	"github.com/aquasecurity/fanal/image"
	"github.com/aquasecurity/fanal/types"
	"github.com/aquasecurity/trivy-db/pkg/db"
	"github.com/aquasecurity/trivy/pkg/detector/ospkg"
	"github.com/aquasecurity/trivy/pkg/rpc/client"
	"github.com/aquasecurity/trivy/pkg/scanner"
	"github.com/aquasecurity/trivy/pkg/scanner/local"
	"github.com/aquasecurity/trivy/pkg/vulnerability"
)

// Injectors from inject.go:

// initializeDockerScanner is for container image scanning in standalone mode
// e.g. dockerd, container registry, podman, etc.
func initializeDockerScanner(ctx context.Context, imageName string, artifactCache cache.ArtifactCache, localArtifactCache cache.LocalArtifactCache, dockerOpt types.DockerOption, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	applierApplier := applier.NewApplier(localArtifactCache)
	detector := ospkg.Detector{}
	config := db.Config{}
	client := vulnerability.NewClient(config)
	localScanner := local.NewScanner(applierApplier, detector, client)
	typesImage, cleanup, err := image.NewContainerImage(ctx, imageName, dockerOpt)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	artifactArtifact, err := image2.NewArtifact(typesImage, artifactCache, artifactOption)
	if err != nil {
		cleanup()
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(localScanner, artifactArtifact)
	return scannerScanner, func() {
		cleanup()
	}, nil
}

// initializeArchiveScanner is for container image archive scanning in standalone mode
// e.g. docker save -o alpine.tar alpine:3.15
func initializeArchiveScanner(ctx context.Context, filePath string, artifactCache cache.ArtifactCache, localArtifactCache cache.LocalArtifactCache, artifactOption artifact.Option) (scanner.Scanner, error) {
	applierApplier := applier.NewApplier(localArtifactCache)
	detector := ospkg.Detector{}
	config := db.Config{}
	client := vulnerability.NewClient(config)
	localScanner := local.NewScanner(applierApplier, detector, client)
	typesImage, err := image.NewArchiveImage(filePath)
	if err != nil {
		return scanner.Scanner{}, err
	}
	artifactArtifact, err := image2.NewArtifact(typesImage, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, err
	}
	scannerScanner := scanner.NewScanner(localScanner, artifactArtifact)
	return scannerScanner, nil
}

// initializeFilesystemScanner is for filesystem scanning in standalone mode
func initializeFilesystemScanner(ctx context.Context, path string, artifactCache cache.ArtifactCache, localArtifactCache cache.LocalArtifactCache, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	applierApplier := applier.NewApplier(localArtifactCache)
	detector := ospkg.Detector{}
	config := db.Config{}
	client := vulnerability.NewClient(config)
	localScanner := local.NewScanner(applierApplier, detector, client)
	artifactArtifact, err := local2.NewArtifact(path, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(localScanner, artifactArtifact)
	return scannerScanner, func() {
	}, nil
}

func initializeRepositoryScanner(ctx context.Context, url string, artifactCache cache.ArtifactCache, localArtifactCache cache.LocalArtifactCache, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	applierApplier := applier.NewApplier(localArtifactCache)
	detector := ospkg.Detector{}
	config := db.Config{}
	client := vulnerability.NewClient(config)
	localScanner := local.NewScanner(applierApplier, detector, client)
	artifactArtifact, cleanup, err := remote.NewArtifact(url, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(localScanner, artifactArtifact)
	return scannerScanner, func() {
		cleanup()
	}, nil
}

func initializeSBOMScanner(ctx context.Context, report types.ArtifactType, path string, artifactCache cache.ArtifactCache, localArtifactCache cache.LocalArtifactCache, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	applierApplier := applier.NewApplier(localArtifactCache)
	detector := ospkg.Detector{}
	config := db.Config{}
	client := vulnerability.NewClient(config)
	localScanner := local.NewScanner(applierApplier, detector, client)
	artifactArtifact, err := sbom2.NewArtifact(report, path, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(localScanner, artifactArtifact)
	return scannerScanner, func() {
	}, nil
}

// initializeRemoteDockerScanner is for container image scanning in client/server mode
// e.g. dockerd, container registry, podman, etc.
func initializeRemoteDockerScanner(ctx context.Context, imageName string, artifactCache cache.ArtifactCache, remoteScanOptions client.ScannerOption, dockerOpt types.DockerOption, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	v := _wireValue
	clientScanner := client.NewScanner(remoteScanOptions, v...)
	typesImage, cleanup, err := image.NewContainerImage(ctx, imageName, dockerOpt)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	artifactArtifact, err := image2.NewArtifact(typesImage, artifactCache, artifactOption)
	if err != nil {
		cleanup()
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(clientScanner, artifactArtifact)
	return scannerScanner, func() {
		cleanup()
	}, nil
}

var (
	_wireValue = []client.Option(nil)
)

// initializeRemoteArchiveScanner is for container image archive scanning in client/server mode
// e.g. docker save -o alpine.tar alpine:3.15
func initializeRemoteArchiveScanner(ctx context.Context, filePath string, artifactCache cache.ArtifactCache, remoteScanOptions client.ScannerOption, artifactOption artifact.Option) (scanner.Scanner, error) {
	v := _wireValue
	clientScanner := client.NewScanner(remoteScanOptions, v...)
	typesImage, err := image.NewArchiveImage(filePath)
	if err != nil {
		return scanner.Scanner{}, err
	}
	artifactArtifact, err := image2.NewArtifact(typesImage, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, err
	}
	scannerScanner := scanner.NewScanner(clientScanner, artifactArtifact)
	return scannerScanner, nil
}

// initializeRemoteFilesystemScanner is for filesystem scanning in client/server mode
func initializeRemoteFilesystemScanner(ctx context.Context, path string, artifactCache cache.ArtifactCache, remoteScanOptions client.ScannerOption, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	v := _wireValue
	clientScanner := client.NewScanner(remoteScanOptions, v...)
	artifactArtifact, err := local2.NewArtifact(path, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(clientScanner, artifactArtifact)
	return scannerScanner, func() {
	}, nil
}

// initializeRemoteCycloneDXScanner is for cycloneDX scanning in client/server mode
func initializeRemoteSBOMScanner(ctx context.Context, report types.ArtifactType, path string, artifactCache cache.ArtifactCache, remoteScanOptions client.ScannerOption, artifactOption artifact.Option) (scanner.Scanner, func(), error) {
	v := _wireValue
	clientScanner := client.NewScanner(remoteScanOptions, v...)
	artifactArtifact, err := sbom2.NewArtifact(report, path, artifactCache, artifactOption)
	if err != nil {
		return scanner.Scanner{}, nil, err
	}
	scannerScanner := scanner.NewScanner(clientScanner, artifactArtifact)
	return scannerScanner, func() {
	}, nil
}
