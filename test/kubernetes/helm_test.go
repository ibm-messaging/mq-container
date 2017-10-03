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
	if !volumesAvailable(t, cs) {
		t.Skipf("Skipping test because no persistent volumes were found")
	}
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

// TestPassThroughValues tests several values which are set when installing
// the Helm chart, and should be passed straight through to Kubernetes
func TestPassThroughValues(t *testing.T) {
	cs := kubeLogin(t)
	release := strings.ToLower(t.Name())
	queueManagerName := "foo"
	requestCPU := "501m"
	requestMem := "501Mi"
	limitCPU := "502m"
	limitMem := "502Mi"
	helmInstall(t, cs, release,
		"license=accept",
		"persistence.enabled=false",
		"resources.requests.cpu="+requestCPU,
		"resources.requests.memory="+requestMem,
		"resources.limits.cpu="+limitCPU,
		"resources.limits.memory="+limitMem,
		"queueManager.name="+queueManagerName,
	)
	defer helmDelete(t, cs, release)
	waitForReady(t, cs, release)
	pods := getPodsForHelmRelease(t, cs, release)
	pod := pods.Items[0]

	t.Run("resources.requests.cpu", func(t *testing.T) {
		cpu := pod.Spec.Containers[0].Resources.Requests.Cpu()
		if cpu.String() != requestCPU {
			t.Errorf("Expected requested CPU to be %v, got %v", requestCPU, cpu.String())
		}
	})
	t.Run("resources.requests.memory", func(t *testing.T) {
		mem := pod.Spec.Containers[0].Resources.Requests.Memory()
		if mem.String() != requestMem {
			t.Errorf("Expected requested memory to be %v, got %v", requestMem, mem.String())
		}
	})
	t.Run("resources.limits.cpu", func(t *testing.T) {
		cpu := pod.Spec.Containers[0].Resources.Limits.Cpu()
		if cpu.String() != limitCPU {
			t.Errorf("Expected CPU limits to be %v, got %v", limitCPU, cpu.String())
		}
	})
	t.Run("resources.limits.memory", func(t *testing.T) {
		mem := pod.Spec.Containers[0].Resources.Limits.Memory()
		if mem.String() != limitMem {
			t.Errorf("Expected memory to be %v, got %v", limitMem, mem.String())
		}
	})
	t.Run("queueManager.name", func(t *testing.T) {
		out, _, err := kubeExec(t, pod.Name, "dspmq", "-n")
		if err != nil {
			t.Fatal(err)
		}
		// Example output of `dspmq -n`:
		// QMNAME(qm1)      STATUS(RUNNING)
		n := strings.Fields(out)[0]
		n = strings.Split(n, "(")[1]
		n = strings.Trim(n, "() ")
		t.Logf("Queue manager name detected: %v", n)
		if n != queueManagerName {
			t.Errorf("Expected queue manager name to be %v, got %v", queueManagerName, n)
		}
	})
}
