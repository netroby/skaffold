/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schema

import (
	"github.com/pkg/errors"

	apiversion "github.com/GoogleContainerTools/skaffold/pkg/skaffold/apiversion"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	misc "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
	"gopkg.in/yaml.v2"
)

type APIVersion struct {
	Version string `yaml:"apiVersion"`
}

var schemaVersions = versions{
	{v1alpha1.Version, v1alpha1.NewSkaffoldPipeline},
	{v1alpha2.Version, v1alpha2.NewSkaffoldPipeline},
	{v1alpha3.Version, v1alpha3.NewSkaffoldPipeline},
	{v1alpha4.Version, v1alpha4.NewSkaffoldPipeline},
	{latest.Version, latest.NewSkaffoldPipeline},
}

type version struct {
	apiVersion string
	factory    func() util.VersionedConfig
}

type versions []version

// Find search the constructor for a given api version.
func (v *versions) Find(apiVersion string) (func() util.VersionedConfig, bool) {
	for _, version := range *v {
		if version.apiVersion == apiVersion {
			return version.factory, true
		}
	}

	return nil, false
}

// ParseConfig reads a configuration file.
func ParseConfig(filename string, applyDefaults bool) (util.VersionedConfig, error) {
	buf, err := misc.ReadConfiguration(filename)
	if err != nil {
		return nil, errors.Wrap(err, "read skaffold config")
	}

	apiVersion := &APIVersion{}
	if err := yaml.Unmarshal(buf, apiVersion); err != nil {
		return nil, errors.Wrap(err, "parsing api version")
	}

	factory, present := schemaVersions.Find(apiVersion.Version)
	if !present {
		return nil, errors.Errorf("unknown api version: '%s'", apiVersion.Version)
	}

	cfg := factory()
	if err := cfg.Parse(buf, applyDefaults); err != nil {
		return nil, errors.Wrap(err, "unable to parse config")
	}

	if err := yamltags.ProcessStruct(cfg); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	return cfg, nil
}

// CheckVersionIsLatest checks that a given version is the most recent.
func CheckVersionIsLatest(apiVersion string) error {
	parsedVersion, err := apiversion.Parse(apiVersion)
	if err != nil {
		return errors.Wrap(err, "parsing api version")
	}

	if parsedVersion.LT(apiversion.MustParse(latest.Version)) {
		return errors.New("config version out of date: run `skaffold fix`")
	}

	if parsedVersion.GT(apiversion.MustParse(latest.Version)) {
		return errors.New("config version is too new for this version of skaffold: upgrade skaffold")
	}

	return nil
}

// UpgradeToLatest upgrades a configuration to the latest version.
func UpgradeToLatest(vc util.VersionedConfig) (util.VersionedConfig, error) {
	var err error

	for vc.GetVersion() != latest.Version {
		vc, err = vc.Upgrade()
		if err != nil {
			return nil, errors.Wrapf(err, "transforming skaffold config")
		}
	}

	return vc, nil
}
