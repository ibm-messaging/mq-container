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
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var namespace string = "default"

// runCommand runs an OS command.  On Linux it waits for the command to
// complete and returns the exit status (return code).
// TODO: duplicated from cmd/runmqserver/main.go
func runCommand(t *testing.T, name string, arg ...string) (string, int, error) {
	t.Logf("Running command %v %v", name, arg)
	cmd := exec.Command(name, arg...)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		var rc int
		// Only works on Linux
		if runtime.GOOS == "linux" {
			// func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error)
			var ws unix.WaitStatus
			//var rusage syscall.Rusage
			unix.Wait4(cmd.Process.Pid, &ws, 0, nil)
			//ee := err.(*os.SyscallError)
			//ws := ee.Sys().(syscall.WaitStatus)
			rc = ws.ExitStatus()
		} else {
			rc = -1
		}
		if rc == 0 {
			return string(out), rc, nil
		}
		return string(out), rc, err
	}
	return string(out), 0, nil
}

func helmDelete(t *testing.T, release string) {
	out, _, err := runCommand(t, "helm", "delete", "--purge", release)
	if err != nil {
		t.Error(out)
		t.Fatal(err)
	}
}

func helmDeleteWithPVC(t *testing.T, cs *kubernetes.Clientset, release string) {
	helmDelete(t, release)
	pvcs, err := cs.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: "app=" + release + "-mq",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, pvc := range pvcs.Items {
		t.Logf("Deleting persistent volume claim: %v", pvc.Name)
		err := cs.CoreV1().PersistentVolumeClaims(namespace).Delete(pvc.Name, &metav1.DeleteOptions{})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func kubeLogin(t *testing.T) *kubernetes.Clientset {
	kc := os.Getenv("HOME") + "/.kube/config"
	c, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := kubernetes.NewForConfig(c)
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

func waitForTerminated(pods types.PodInterface, name string) (v1.PodPhase, error) {
	status := v1.PodUnknown
	var pod *v1.Pod
	var err error
	for status != v1.PodSucceeded && status != v1.PodFailed {
		pod, err = pods.Get(name, metav1.GetOptions{})
		if err != nil {
			return status, err
		}
		status = pod.Status.Phase
		time.Sleep(1 * time.Second)
	}
	// The LastTerminationState doesn't seem to include an exit code
	//t := pod.Status.ContainerStatuses[0].LastTerminationState
	return status, nil
}

func TestHelmGoldenPath(t *testing.T) {
	cs := kubeLogin(t)
	chart := "../charts/mqadvanced-dev"
	image := "master.cfc:8500/default/mqadvanced"
	tag := "latest"
	release := strings.ToLower(t.Name())
	out, _, err := runCommand(t, "helm", "install", chart, "--name", release, "--set", "license=accept", "--set", "image.repository="+image, "--set", "image.tag="+tag)
	if err != nil {
		t.Error(out)
		t.Fatal(err)
	}
	defer helmDeleteWithPVC(t, cs, release)

	nodes, err := cs.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("There are %d nodes in the cluster\n", len(nodes.Items))

	pods, err := cs.CoreV1().Pods("").List(metav1.ListOptions{
		LabelSelector: "app=" + release + "-mq",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("There are %d pods with the right label in the cluster\n", len(pods.Items))
}
