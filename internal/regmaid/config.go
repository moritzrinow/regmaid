package regmaid

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Registries  []Registry `yaml:"registries"`
	Policies    []Policy   `yaml:"policies"`
	DockerCreds bool       `yaml:"dockerCreds"`
}

type Registry struct {
	Name                  string `yaml:"name"`
	Host                  string `yaml:"host"`
	Insecure              bool   `yaml:"insecure"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	Token                 string `yaml:"token"`
	SkipTlsVerify         bool   `yaml:"skipTlsVerify"`
	ClientCert            string `yaml:"clientCert"`
	ClientKey             string `yaml:"clientKey"`
	RegCert               string `yaml:"regCert"`
	MaxConcurrentRequests int    `yaml:"maxConcurrentRequests"`
	MaxRequestsPerSecond  int    `yaml:"maxRequestsPerSecond"`
}

type Policy struct {
	Name        string `yaml:"name"`
	Registry    string `yaml:"registry"`
	Repository  string `yaml:"repository"`
	Match       string `yaml:"match"`
	Regex	    bool   `yaml:"regex"`
	Keep        int    `yaml:"keep"`
	Retention   string `yaml:"retention"`
	Force       bool   `yaml:"force"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("error reading file from %q: %v", path, err)
	}

	var cfg Config

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	for i := range cfg.Registries {
		r := &cfg.Registries[i]

		if user, exists := os.LookupEnv(fmt.Sprintf("%s_USERNAME", strings.ToUpper(r.Name))); exists {
			r.Username = user
		}
		if password, exists := os.LookupEnv(fmt.Sprintf("%s_PASSWORD", strings.ToUpper(r.Name))); exists {
			r.Password = password
		}
		if token, exists := os.LookupEnv(fmt.Sprintf("%s_TOKEN", strings.ToUpper(r.Name))); exists {
			r.Token = token
		}
		if key, exists := os.LookupEnv(fmt.Sprintf("%s_CLIENTKEY", strings.ToUpper(r.Name))); exists {
			r.ClientKey = key
		}
	}

	if err := cfg.Validate(); err != nil {
		return &cfg, fmt.Errorf("invalid config: %v", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	for i, reg := range c.Registries {
		if len(reg.Name) == 0 {
			return fmt.Errorf("registry %d has no name specified", i)
		}

		if len(reg.Host) == 0 {
			return fmt.Errorf("registry %q has no host specified", reg.Name)
		}
	}

	for i, policy := range c.Policies {
		if len(policy.Name) == 0 {
			return fmt.Errorf("policy %d has no name specified", i)
		}

		if len(policy.Registry) == 0 {
			return fmt.Errorf("policy %q has no registry specified", policy.Name)
		}

		if len(policy.Repository) == 0 {
			return fmt.Errorf("policy %q has no repository specified", policy.Name)
		}

		registryExists := slices.ContainsFunc(c.Registries, func(r Registry) bool {
			return r.Name == policy.Registry
		})

		if !registryExists {
			return fmt.Errorf("policy %q specified registry %q but no such registry was defined", policy.Name, policy.Registry)
		}

		if policy.Keep <= 0 && !policy.Force {
			return fmt.Errorf("policy %q must specify property 'keep' with value greater than 0 or set 'force' to 'true'", policy.Name)
		}
	}

	return nil
}
