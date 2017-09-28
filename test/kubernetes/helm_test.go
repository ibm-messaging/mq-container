/*
© Copyright IBM Corporation 2017

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
package main

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var namespace = "default"

// Prior to running this test, a Persistent Volume must be created
func TestHelmPredefinedVolume(t *testing.T) {
	cs := kubeLogin(t)
	release := strings.ToLower(t.Name())
	helmInstall(t, cs, release, "license=accept", "persistence.useDynamicProvisioning=false")
	defer helmDelete(t, cs, release)
	defer helmDeletePVC(t, cs, release)
	waitForReady(t, cs, release)
}

func TestHelmStorageClass(t *testing.T) {
	cs := kubeLogin(t)
	release := strings.ToLower(t.Name())
	if !storageClassesDefined(t, cs) {
		t.Skipf("Skipping test because no storage classes were found")
	}
	helmInstall(t, cs, release, "license=accept", "persistence.useDynamicProvisioning=true")
	defer helmDelete(t, cs, release)
	defer helmDeletePVC(t, cs, release)
	waitForReady(t, cs, release)
}

func TestPersistenceDisabled(t *testing.T) {
	cs := kubeLogin(t)
	release := strings.ToLower(t.Name())
	helmInstall(t, cs, release, "license=accept", "persistence.enabled=false")
	defer helmDelete(t, cs, release)
	waitForReady(t, cs, release)

	// Check that no PVCs were created
	pvcs, err := cs.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: "release=" + release,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pvcs.Items) > 0 {
		t.Errorf("Expected no PVC, found %v (%+v)", len(pvcs.Items), pvcs.Items)
	}
}
