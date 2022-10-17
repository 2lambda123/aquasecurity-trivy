package vm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	ebsfile "github.com/masahiro331/go-ebs-file"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"

	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/fanal/artifact"
	"github.com/aquasecurity/trivy/pkg/fanal/cache"
	"github.com/aquasecurity/trivy/pkg/fanal/handler"
	"github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/fanal/vm/storage"
	"github.com/aquasecurity/trivy/pkg/fanal/walker"
	"github.com/aquasecurity/trivy/pkg/log"
)

const (
	parallel       = 5
	cacheSize      = 40 << 20 // 40 MB
	cacheKeyPrefix = "vm"
)

type Artifact struct {
	filePath       string
	cache          cache.ArtifactCache
	analyzer       analyzer.AnalyzerGroup
	handlerManager handler.Manager
	walker         walker.VM

	artifactOption artifact.Option
}

func (a Artifact) Inspect(ctx context.Context) (reference types.ArtifactReference, err error) {
	result := analyzer.NewAnalysisResult()

	lruCache, err := lru.New(cacheSize)
	if err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to create new lru cache: %w", err)
	}
	defer lruCache.Purge()

	s, err := storage.NewStorage(a.filePath, ebsfile.Option{}, ctx, lruCache)
	if err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to new storage: %w", err)
	}
	defer s.Close()

	sr, cacheKey, err := s.Open(a.filePath)
	if err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to open storage: %w", err)
	}
	cacheKey = vmCacheKey(cacheKey)

	missingVMCache, _, err := a.cache.MissingBlobs(cacheKey, []string{cacheKey})
	if err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to missing blobs from cache: %w", err)
	}
	if missingVMCache {
		log.Logger.Debugf("Missing virtual machine cache: %s", cacheKey)
	} else {
		return types.ArtifactReference{
			Name:    a.filePath,
			Type:    types.ArtifactVM,
			ID:      cacheKey, // use a cache key as pseudo artifact ID
			BlobIDs: []string{cacheKey},
		}, nil
	}

	var wg sync.WaitGroup
	limit := semaphore.NewWeighted(parallel)

	// TODO: Always walk from the root directory. Consider whether there is a need to be able to set optional
	err = a.walker.Walk(sr, lruCache, "/", func(filePath string, info os.FileInfo, opener analyzer.Opener) error {
		opts := analyzer.AnalysisOptions{Offline: a.artifactOption.Offline}
		// Skip special files
		// 	0x1000:	S_IFIFO (FIFO)
		// 	0x2000:	S_IFCHR (Character device)
		// 	0x6000:	S_IFBLK (Block device)
		// 	0xA000:	S_IFLNK (Symbolic link)
		// 	0xC000:	S_IFSOCK (Socket)
		if info.Mode()&0x1000 == 0x1000 ||
			info.Mode()&0x2000 == 0x2000 ||
			info.Mode()&0x6000 == 0x6000 ||
			info.Mode()&0xA000 == 0xA000 ||
			info.Mode()&0xc000 == 0xc000 {
			return nil
		}
		path := strings.TrimPrefix(filePath, "/")
		if err = a.analyzer.AnalyzeFile(ctx, &wg, limit, result, "/", path, info, opener, nil, opts); err != nil {
			return xerrors.Errorf("analyze file (%s): %w", path, err)
		}
		return nil
	})
	if err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("walk vm error: %w", err)
	}
	result.Sort()

	blobInfo := types.BlobInfo{
		SchemaVersion:   types.BlobJSONSchemaVersion,
		OS:              result.OS,
		Repository:      result.Repository,
		PackageInfos:    result.PackageInfos,
		Applications:    result.Applications,
		Secrets:         result.Secrets,
		Licenses:        result.Licenses,
		CustomResources: result.CustomResources,
	}

	if err = a.handlerManager.PostHandle(ctx, result, &blobInfo); err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to call hooks: %w", err)
	}

	if err = a.cache.PutBlob(cacheKey, blobInfo); err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to store blob (%s) in cache: %w", cacheKey, err)
	}
	info := types.ArtifactInfo{
		SchemaVersion: types.ArtifactJSONSchemaVersion,
	}
	if err = a.cache.PutArtifact(cacheKey, info); err != nil {
		return types.ArtifactReference{}, xerrors.Errorf("failed to put image info into the cache: %w", err)
	}

	return types.ArtifactReference{
		Name:    a.filePath,
		Type:    types.ArtifactVM,
		ID:      cacheKey, // use a cache key as pseudo artifact ID
		BlobIDs: []string{cacheKey},
	}, nil
}

func (a Artifact) Clean(_ types.ArtifactReference) error {
	return nil
}

func NewArtifact(filePath string, c cache.ArtifactCache, opt artifact.Option) (artifact.Artifact, error) {
	handlerManager, err := handler.NewManager(opt)
	if err != nil {
		return nil, xerrors.Errorf("handler init error: %w", err)
	}
	a, err := analyzer.NewAnalyzerGroup(analyzer.AnalyzerOptions{
		Group:               opt.AnalyzerGroup,
		FilePatterns:        opt.FilePatterns,
		DisabledAnalyzers:   opt.DisabledAnalyzers,
		SecretScannerOption: opt.SecretScannerOption,
	})
	if err != nil {
		return nil, xerrors.Errorf("analyzer group error: %w", err)
	}

	return Artifact{
		filePath:       filepath.Clean(filePath),
		cache:          c,
		handlerManager: handlerManager,
		analyzer:       a,
		walker:         walker.NewVM(opt.SkipFiles, opt.SkipDirs),

		artifactOption: opt,
	}, nil
}

func vmCacheKey(key string) string {
	return fmt.Sprintf("%s:%s", cacheKeyPrefix, key)
}
