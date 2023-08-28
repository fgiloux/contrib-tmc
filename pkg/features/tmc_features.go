/*
Copyright 2022 The KCP Authors.

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

package features

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/pflag"

	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
	logsapi "k8s.io/component-base/logs/api/v1"
)

const (
	// Every feature gate should add method here following this template:
	//
	// // owner: @username
	// // alpha: v1.4
	// MyFeature() bool.

	// owner: @sttts
	// alpha: v0.4
	//
	// Enable the scheduling.kcp.io/v1alpha1 API group, and related controllers.
	LocationAPI featuregate.Feature = "KCPLocationAPI"

	// owner: @aojea
	// alpha: v0.8
	//
	// Enable reverse tunnels to the downstream clusters through the syncers.
	SyncerTunnel featuregate.Feature = "KCPSyncerTunnel"
)

// DefaultFeatureGate exposes the upstream feature gate, but with our gate setting applied.
var DefaultFeatureGate = utilfeature.DefaultFeatureGate

func KnownFeatures() []string {
	features := make([]string, 0, len(defaultFeatureGates))
	for k := range defaultFeatureGates {
		features = append(features, string(k))
	}
	return features
}

// NewFlagValue returns a wrapper to be used for a pflag flag value.
func NewFlagValue() pflag.Value {
	return &kcpFeatureGate{
		utilfeature.DefaultMutableFeatureGate,
	}
}

type kcpFeatureGate struct {
	featuregate.MutableFeatureGate
}

func (f *kcpFeatureGate) String() string {
	pairs := []string{}
	for k, v := range defaultFeatureGates {
		pairs = append(pairs, fmt.Sprintf("%s=%t", k, v.Default))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

func (f *kcpFeatureGate) Type() string {
	return "mapStringBool"
}

// defaultFeatureGates consists of all known feature keys.
// To add a new feature, define a key for it above and add it
// here. The features will be available throughout binaries.
var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	LocationAPI:               {Default: true, PreRelease: featuregate.Alpha},
	SyncerTunnel:              {Default: true, PreRelease: featuregate.Alpha},
	logsapi.ContextualLogging: {Default: true, PreRelease: featuregate.Alpha},
}
