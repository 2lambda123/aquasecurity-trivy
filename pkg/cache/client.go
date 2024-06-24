package cache

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

const (
	TypeUnknown Type = "unknown"
	TypeFS      Type = "fs"
	TypeRedis   Type = "redis"
)

type Type string

type Options struct {
	Backend     string
	CacheDir    string
	RedisCACert string
	RedisCert   string
	RedisKey    string
	RedisTLS    bool
	TTL         time.Duration
}

func NewType(backend string) Type {
	// "redis://" or "fs" are allowed for now
	// An empty value is also allowed for testability
	switch {
	case strings.HasPrefix(backend, "redis://"):
		return TypeRedis
	case backend == "fs", backend == "":
		return TypeFS
	default:
		return TypeUnknown
	}
}

// New returns a new cache client
func New(opts Options) (Cache, func(), error) {
	cleanup := func() {} // To avoid panic

	var cache Cache
	t := NewType(opts.Backend)
	switch t {
	case TypeRedis:
		redisCache, err := NewRedisCache(opts.Backend, opts.RedisCACert, opts.RedisCert, opts.RedisKey, opts.RedisTLS, opts.TTL)
		if err != nil {
			return nil, cleanup, xerrors.Errorf("unable to initialize redis cache: %w", err)
		}
		cache = redisCache
	case TypeFS:
		// standalone mode
		fsCache, err := NewFSCache(opts.CacheDir)
		if err != nil {
			return nil, cleanup, xerrors.Errorf("unable to initialize fs cache: %w", err)
		}
		cache = fsCache
	default:
		return nil, cleanup, xerrors.Errorf("unknown cache type: %s", t)
	}
	return cache, func() { _ = cache.Close() }, nil
}

// GetTLSConfig gets tls config from CA, Cert and Key file
func GetTLSConfig(caCertPath, certPath, keyPath string) (*x509.CertPool, tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, tls.Certificate{}, err
	}

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, tls.Certificate{}, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return caCertPool, cert, nil
}
