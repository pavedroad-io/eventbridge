package s3

import "fmt"

// Provider for creating new client
type Provider struct {
	// Name AWS, GCP, etc
	Name string `yaml:"name" json:"name"`

	// Credentials key name
	Credentials string `yaml:"credentials" json:"credentials"`
	// Key value
	Key string `yaml:"key" json:"key"`

	// Region us-west-1, etc
	Region string `yaml:"region" json:"region"`

	// Endpoint s3.aws.com
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

type Providers []Provider

func (ps *Providers) Lookup(pName string) (Provider, error) {
	rp := Provider{}
	for _, p := range *ps {
		if p.Name == pName {
			return p, nil
		}
	}

	return rp, fmt.Errorf("Provider lookup failed for %v\n", pName)
}
