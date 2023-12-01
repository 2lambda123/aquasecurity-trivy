package config

import (
	"github.com/aquasecurity/defsec/pkg/providers/aws/config"
	"github.com/aquasecurity/trivy/pkg/iac/scanners/cloudformation/parser"
)

// Adapt ...
func Adapt(cfFile parser.FileContext) config.Config {
	return config.Config{
		ConfigurationAggregrator: getConfigurationAggregator(cfFile),
	}
}
