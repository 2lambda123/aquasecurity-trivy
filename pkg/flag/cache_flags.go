package flag

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

// e.g. config yaml
// cache:
//   clear: true
//   backend: "redis://localhost:6379"
//   redis:
//    ca: ca-cert.pem
//    cert: cert.pem
//    key: key.pem
var (
	ClearCacheFlag = Flag{
		Name:       "clear-cache",
		ConfigName: "cache.clear",
		Value:      false,
		Usage:      "clear image caches without scanning",
	}
	CacheBackendFlag = Flag{
		Name:       "cache-backend",
		ConfigName: "cache.backend",
		Value:      "fs",
		Usage:      "cache backend (e.g. redis://localhost:6379)",
	}
	CacheTTLFlag = Flag{
		Name:       "cache-ttl",
		ConfigName: "cache.ttl",
		Value:      time.Duration(0),
		Usage:      "cache TTL when using redis as cache backend",
	}
	RedisCACertFlag = Flag{
		Name:       "redis-ca",
		ConfigName: "cache.redis.ca",
		Value:      "",
		Usage:      "redis ca file location, if using redis as cache backend",
	}
	RedisCertFlag = Flag{
		Name:       "redis-cert",
		ConfigName: "cache.redis.cert",
		Value:      "",
		Usage:      "redis certificate file location, if using redis as cache backend",
	}
	RedisKeyFlag = Flag{
		Name:       "redis-key",
		ConfigName: "cache.redis.key",
		Value:      "",
		Usage:      "redis key file location, if using redis as cache backend",
	}
)

// CacheFlagGroup composes common printer flag structs used for commands requiring cache logic.
type CacheFlagGroup struct {
	ClearCache   *Flag
	CacheBackend *Flag
	CacheTTL     *Flag

	RedisCACert *Flag
	RedisCert   *Flag
	RedisKey    *Flag
}

type CacheOptions struct {
	ClearCache   bool
	CacheBackend string
	CacheTTL     time.Duration
	RedisOptions
}

// RedisOptions holds the options for redis cache
type RedisOptions struct {
	RedisCACert string
	RedisCert   string
	RedisKey    string
}

// NewCacheFlagGroup returns a default CacheFlagGroup
func NewCacheFlagGroup() *CacheFlagGroup {
	return &CacheFlagGroup{
		ClearCache:   lo.ToPtr(ClearCacheFlag),
		CacheBackend: lo.ToPtr(CacheBackendFlag),
		CacheTTL:     lo.ToPtr(CacheTTLFlag),
		RedisCACert:  lo.ToPtr(RedisCACertFlag),
		RedisCert:    lo.ToPtr(RedisCertFlag),
		RedisKey:     lo.ToPtr(RedisKeyFlag),
	}
}

func (f *CacheFlagGroup) flags() []*Flag {
	return []*Flag{f.ClearCache, f.CacheBackend, f.CacheTTL, f.RedisCACert, f.RedisCert, f.RedisKey}
}

func (f *CacheFlagGroup) AddFlags(cmd *cobra.Command) {
	for _, flag := range f.flags() {
		addFlag(cmd, flag)
	}
}

func (f *CacheFlagGroup) Bind(cmd *cobra.Command) error {
	for _, flag := range f.flags() {
		if err := bind(cmd, flag); err != nil {
			return err
		}
	}
	return nil
}

func (f *CacheFlagGroup) ToOptions() (CacheOptions, error) {
	cacheBackend := getString(f.CacheBackend)
	redisOptions := RedisOptions{
		RedisCACert: getString(f.RedisCACert),
		RedisCert:   getString(f.RedisCert),
		RedisKey:    getString(f.RedisKey),
	}

	// "redis://" or "fs" are allowed for now
	// An empty value is also allowed for testability
	if !strings.HasPrefix(cacheBackend, "redis://") &&
		cacheBackend != "fs" && cacheBackend != "" {
		return CacheOptions{}, xerrors.Errorf("unsupported cache backend: %s", cacheBackend)
	}
	// if one of redis option not nil, make sure CA, cert, and key provided
	if !lo.IsEmpty(redisOptions) {
		if redisOptions.RedisCACert == "" || redisOptions.RedisCert == "" || redisOptions.RedisKey == "" {
			return CacheOptions{}, xerrors.Errorf("you must provide Redis CA, cert and key file path when using TLS")
		}
	}

	return CacheOptions{
		ClearCache:   getBool(f.ClearCache),
		CacheBackend: cacheBackend,
		CacheTTL:     getDuration(f.CacheTTL),
		RedisOptions: redisOptions,
	}, nil
}

// CacheBackendMasked returns the redis connection string masking credentials
func (o *CacheOptions) CacheBackendMasked() string {
	endIndex := strings.Index(o.CacheBackend, "@")
	if endIndex == -1 {
		return o.CacheBackend
	}

	startIndex := strings.Index(o.CacheBackend, "//")

	return fmt.Sprintf("%s****%s", o.CacheBackend[:startIndex+2], o.CacheBackend[endIndex:])
}
