package regmaid

import (
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
)

func NewRegistryClient(c *Config) *regclient.RegClient {
	opts := make([]regclient.Opt, 0)

	for _, r := range c.Registries {
		host := config.Host{
			Name:          r.Host,
			User:          r.Username,
			Pass:          r.Password,
			TLS:           getTlsConfig(&r),
			Token:         r.Token,
			RegCert:       r.RegCert,
			ClientCert:    r.ClientCert,
			ClientKey:     r.ClientKey,
			ReqConcurrent: int64(r.MaxConcurrentRequests),
			ReqPerSec:     float64(r.MaxRequestsPerSecond),
		}

		opts = append(opts, regclient.WithConfigHost(host))
	}

	if c.DockerCreds {
		opts = append(opts, regclient.WithDockerCreds())
	}

	return regclient.New(opts...)
}

func getTlsConfig(r *Registry) config.TLSConf {
	if r.Insecure {
		return config.TLSDisabled
	}

	if r.SkipTlsVerify {
		return config.TLSInsecure
	}

	return config.TLSEnabled
}
