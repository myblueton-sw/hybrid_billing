package server

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
)

const (
	pluginLicensesSetting = "plugin_licenses"
	licenseStatePermitted = "permitted"
	licenseStateRefused   = "not_permitted"
	sampleLicense         = "Apache-2.0"
	sampleClosedLicense   = "LicenseRef-sample-proprietary"
)

type PluginSettings struct {
	Raw   string
	Given bool
	Uses  int
}

func DefaultPluginLicenses() []string {
	return []string{"Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC", "LGPL-2.1-only", "LGPL-3.0-only", "MIT", "MPL-2.0", "Zlib"}
}

func (p PluginSettings) licenses() ([]string, error) {
	switch {
	case p.Uses > 1:
		return nil, settingError(pluginLicensesSetting)
	case !p.Given:
		return DefaultPluginLicenses(), nil
	case p.Raw == "":
		return []string{}, nil
	}
	list := strings.Split(p.Raw, ",")
	sort.Strings(list)
	if !plugins.ValidLicenses(list) {
		return nil, settingError(pluginLicensesSetting)
	}
	return list, nil
}

func (p PluginSettings) source() string {
	if p.Given {
		return SourceStartFlag
	}
	return SourceDefault
}

func sampleLicensed(seed string, licenses []string) error {
	if seed == SeedSample && !slices.Contains(licenses, sampleLicense) {
		return fmt.Errorf("%w: the sample seed needs %s in %s", ErrConfig, sampleLicense, pluginLicensesSetting)
	}
	return nil
}
