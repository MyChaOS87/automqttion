package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrInvalidTLSConfig indicates the MQTT TLS configuration is invalid or incomplete.
var ErrInvalidTLSConfig = errors.New("invalid mqtt tls configuration")

// TLSConfig describes optional TLS verification and mutual-TLS client certificate files.
type TLSConfig struct {
	Enabled            bool
	CAFile             string
	CertFile           string
	KeyFile            string
	ServerName         string
	InsecureSkipVerify bool
}

// BuildTLSConfig constructs a *tls.Config enforcing TLS 1.2+ and loading optional CA and client certificates.
func BuildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	if (cfg.CertFile != "") != (cfg.KeyFile != "") {
		return nil, fmt.Errorf("%w: both certFile and keyFile must be provided together", ErrInvalidTLSConfig)
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.ServerName,
		//nolint:gosec // G402: InsecureSkipVerify is explicitly configurable for trusted local/self-signed brokers.
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CAFile != "" {
		pemBytes, err := os.ReadFile(filepath.Clean(cfg.CAFile))
		if err != nil {
			return nil, fmt.Errorf("%w: read CA file %q: %w", ErrInvalidTLSConfig, cfg.CAFile, err)
		}

		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}

		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("%w: parse CA certificates from %q", ErrInvalidTLSConfig, cfg.CAFile)
		}

		tlsCfg.RootCAs = pool
	}

	if cfg.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(filepath.Clean(cfg.CertFile), filepath.Clean(cfg.KeyFile))
		if err != nil {
			return nil, fmt.Errorf("%w: load client key pair: %w", ErrInvalidTLSConfig, err)
		}

		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}
