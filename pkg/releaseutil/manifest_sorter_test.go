/*
Copyright The Helm Authors.

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

package releaseutil

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
)

func TestSortManifests(t *testing.T) {
	hookReplicaSetTwo := release.Hook{
		Name:           "second",
		Kind:           "ReplicaSet",
		Path:           "two",
		Weight:         0,
		Events:         []release.HookEvent{release.HookPostInstall},
		DeletePolicies: []release.HookDeletePolicy{},
		Manifest: `kind: ReplicaSet
apiVersion: v1beta1
metadata:
  name: second
  annotations:
    "helm.sh/hook": post-install`,
	}

	// This should be skipped because "helm.sh/hook": no-such-hook
	unknownHookManifest := `kind: ReplicaSet
apiVersion: v1beta1
metadata:
  name: third
  annotations:
    "helm.sh/hook": no-such-hook`

	manifestPodFour := Manifest{
		Name:   "four",
		Weight: 0,
		Head: &SimpleHead{
			Kind:    "Pod",
			Version: "v1",
		},
		Content: `kind: Pod
apiVersion: v1
metadata:
  name: fourth
  annotations:
    nothing: here`,
	}

	hookReplicaSetFive := release.Hook{
		Name:           "fifth",
		Kind:           "ReplicaSet",
		Path:           "five",
		Weight:         0,
		Events:         []release.HookEvent{release.HookPostDelete, release.HookPostInstall},
		DeletePolicies: []release.HookDeletePolicy{},
		Manifest: `kind: ReplicaSet
apiVersion: v1beta1
metadata:
  name: fifth
  annotations:
    "helm.sh/hook": post-delete, post-install`,
	}

	manifestConfigMapEight := Manifest{
		Name:   "eight",
		Weight: 0,
		Head: &SimpleHead{
			Kind:    "ConfigMap",
			Version: "v1",
		},
		Content: `kind: ConfigMap
apiVersion: v1
metadata:
  name: eighth
data:
  name: value`,
	}

	hookPodEight := release.Hook{
		Name:           "example-test",
		Kind:           "Pod",
		Path:           "eight",
		Weight:         0,
		Events:         []release.HookEvent{release.HookTest},
		DeletePolicies: []release.HookDeletePolicy{},
		Manifest: `apiVersion: v1
kind: Pod
metadata:
  name: example-test
  annotations:
    "helm.sh/hook": test`,
	}

	manifestUnkownNine := Manifest{
		Name:   "nine",
		Weight: 0,
		Head: &SimpleHead{
			Kind:    "Unknown",
			Version: "v1",
		},
		Content: `kind: Unknown
apiVersion: v1
metadata:
  name: ninth
data:
  name: value`,
	}

	manifestUnkownTenWeightMinus1 := Manifest{
		Name:   "ten",
		Weight: -1,
		Head: &SimpleHead{
			Kind:    "Unknown",
			Version: "v1",
		},
		Content: `kind: Unknown
apiVersion: v1
metadata:
  name: tenth
  annotations:
    "helm.sh/weight": -1
data:
  name: value`,
	}

	manifestUnkownElevenWeightPlus1 := Manifest{
		Name:   "eleven",
		Weight: 1,
		Head: &SimpleHead{
			Kind:    "Unknown",
			Version: "v1",
		},
		Content: `kind: Unknown
apiVersion: v1
metadata:
  name: eleventh
  annotations:
    "helm.sh/weight": 1
data:
  name: value`,
	}

	input := map[string]string{
		hookReplicaSetTwo.Path:  hookReplicaSetTwo.Manifest,
		"unkownHook":            unknownHookManifest,
		manifestPodFour.Name:    manifestPodFour.Content,
		hookReplicaSetFive.Path: hookReplicaSetFive.Manifest,

		// Regression test: files with an underscore in the base name should be skipped.
		"six/_six": "invalid manifest",

		// Regression test: files with no content should be skipped.
		"seven": "",

		"eight": fmt.Sprintf("%s\n---\n%s", manifestConfigMapEight.Content, hookPodEight.Manifest),

		manifestUnkownNine.Name:              manifestUnkownNine.Content,
		manifestUnkownTenWeightMinus1.Name:   manifestUnkownTenWeightMinus1.Content,
		manifestUnkownElevenWeightPlus1.Name: manifestUnkownElevenWeightPlus1.Content,
	}

	// Install Order
	hooks, manifests, err := SortManifests(input, chartutil.VersionSet{"v1", "v1beta1"}, InstallOrder)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	expectedHooks := []release.Hook{hookPodEight, hookReplicaSetFive, hookReplicaSetTwo}
	assertMatchingHooks(t, hooks, expectedHooks)

	expectedManifests := []Manifest{manifestUnkownTenWeightMinus1, manifestConfigMapEight, manifestPodFour, manifestUnkownNine, manifestUnkownElevenWeightPlus1}
	assertMatchingManifests(t, manifests, expectedManifests)

	// Uninstall order
	hooksUninstall, manifestsUninstall, err := SortManifests(input, chartutil.VersionSet{"v1", "v1beta1"}, UninstallOrder)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	expectedHooksUninstall := []release.Hook{hookReplicaSetFive, hookReplicaSetTwo, hookPodEight}
	assertMatchingHooks(t, hooksUninstall, expectedHooksUninstall)

	expectedManifestsUninstall := []Manifest{manifestUnkownElevenWeightPlus1, manifestPodFour, manifestConfigMapEight, manifestUnkownNine, manifestUnkownTenWeightMinus1}
	assertMatchingManifests(t, manifestsUninstall, expectedManifestsUninstall)
}

func assertMatchingHooks(t *testing.T, actualHooks []*release.Hook, expectedHooks []release.Hook) {
	maxIdx := int(math.Max(float64(len(actualHooks)), float64(len(expectedHooks))))
	for idx := 0; idx < maxIdx; idx++ {
		if idx >= len(actualHooks) {
			t.Fatalf("Missing hook at index %d : expecting:\n %#v", idx, expectedHooks[idx])
		}
		if idx >= len(expectedHooks) {
			t.Fatalf("Unexpected hook at index %d: \n %#v", idx, &actualHooks[idx])
		}
		if (actualHooks[idx].Name != expectedHooks[idx].Name) ||
			(actualHooks[idx].Kind != expectedHooks[idx].Kind) ||
			(actualHooks[idx].Path != expectedHooks[idx].Path) ||
			(actualHooks[idx].Weight != expectedHooks[idx].Weight) ||
			(actualHooks[idx].Manifest != expectedHooks[idx].Manifest) ||
			!reflect.DeepEqual(actualHooks[idx].Events, expectedHooks[idx].Events) ||
			!reflect.DeepEqual(actualHooks[idx].DeletePolicies, expectedHooks[idx].DeletePolicies) {

			t.Fatalf("Mismatcing hook at index %d - expecting:\n %#v \nbut was:\n %#v", idx, expectedHooks[idx], *actualHooks[idx])
		}
	}
}

func assertMatchingManifests(t *testing.T, actualManifests []Manifest, expectedManifests []Manifest) {
	maxIdx := int(math.Max(float64(len(actualManifests)), float64(len(expectedManifests))))
	for idx := 0; idx < maxIdx; idx++ {
		if idx >= len(actualManifests) {
			t.Fatalf("Missing manifest at index %d : expecting:\n %#v", idx, expectedManifests[idx])
		}
		if idx >= len(expectedManifests) {
			t.Fatalf("Unexpected manifest at index %d: \n %#v", idx, &actualManifests[idx])
		}
		if (actualManifests[idx].Name != expectedManifests[idx].Name) ||
			(actualManifests[idx].Content != expectedManifests[idx].Content) ||
			(actualManifests[idx].Weight != expectedManifests[idx].Weight) ||
			(actualManifests[idx].Head.Kind != expectedManifests[idx].Head.Kind) ||
			(actualManifests[idx].Head.Version != expectedManifests[idx].Head.Version) {

			t.Fatalf("Mismatcing manifest at index %d - expecting:\n %#v \nbut was:\n %#v", idx, expectedManifests[idx], actualManifests[idx])
		}
	}
}
