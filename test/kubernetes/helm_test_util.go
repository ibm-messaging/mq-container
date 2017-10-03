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
	"bytes"
	"fmt"
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

func imageName() string {
	image, ok := os.LookupEnv("TEST_IMAGE")
	if !ok {
		image = "ibmcom/mq"
	}
	return image
}

// runCommand runs an OS command.  On Linux it waits for the command to
// complete and returns the exit status (return code).
// TODO: duplicated from cmd/runmqserver/main.go
func runCommand(t *testing.T, name string, arg ...string) (string, int, error) {
	t.Logf("Running command: %v %v", name, strings.Trim(fmt.Sprintf("%v", arg), "[]"))
	cmd := exec.Command(name, arg...)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		var rc int
		// Only works on Linux
		if runtime.GOOS == "linux" {
			var ws unix.WaitStatus
			unix.Wait4(cmd.Process.Pid, &ws, 0, nil)
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

func inspectLogs(t *testing.T, cs *kubernetes.Clientset, release string) string {
	pods := getPodsForHelmRelease(t, cs, release)
	opt := v1.PodLogOptions{}
	r := cs.CoreV1().Pods(namespace).GetLogs(pods.Items[0].Name, &opt)
	buf := new(bytes.Buffer)
	rc, err := r.Stream()
	if err != nil {
		t.Fatal(err)
	}
	buf.ReadFrom(rc)
	return buf.String()
}

func helmInstall(t *testing.T, cs *kubernetes.Clientset, release string, values ...string) {
	chart := "../../charts/ibm-mqadvanced-server-prod"
	tag := "latest"
	arg := []string{
		"install",
		"--debug",
		chart,
		"--name",
		release,
		"--set",
		"image.repository=" + imageName(),
		"--set",
		"image.tag=" + tag,
		"--set",
		"image.pullSecret=admin.registrykey",
	}
	// Add any extra values to the Helm command
	for _, value := range values {
		arg = append(arg, "--set", value)
	}
	out, _, err := runCommand(t, "helm", arg...)
	t.Log(out)
	if err != nil {
		t.Error(out)
		t.Fatal(err)
	}
}

func helmDelete(t *testing.T, cs *kubernetes.Clientset, release string) {
	t.Log("Deleting Helm release")
	t.Log(inspectLogs(t, cs, release))
	out, _, err := runCommand(t, "helm", "delete", "--purge", release)
	if err != nil {
		t.Error(out)
		t.Fatal(err)
	}
}

func helmDeletePVC(t *testing.T, cs *kubernetes.Clientset, release string) {
	pvcs, err := cs.CoreV1().PersistentVolumeClaims(namespace).List(metav1.ListOptions{
		LabelSelector: "release=" + release,
	})
	if err != nil {
		t.Error(err)
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

func kubeExec(t *testing.T, podName string, name string, arg ...string) (string, int, error) {
	// Current version of Kubernetes Go client doesn't support "exec", so run this via the command line
	param := []string{"exec", podName, "--", name}
	param = append(param, arg...)
	return runCommand(t, "kubectl", param...)
}

func waitForReady(t *testing.T, cs *kubernetes.Clientset, release string) {
	pods := getPodsForHelmRelease(t, cs, release)
	if len(pods.Items) != 1 {
		t.Fatalf("Expected 1 pod, found %v", len(pods.Items))
	}
	pod := pods.Items[0]
	podName := pod.Name
	// Wait for the queue manager container to be started
	running := false
	for !running {
		pod, err := cs.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(pod.Status.ContainerStatuses) > 0 {
			state := pod.Status.ContainerStatuses[0].State
			switch {
			case state.Waiting != nil:
				t.Logf("Waiting for container")
				time.Sleep(1 * time.Second)
			case state.Running != nil:
				running = true
			}
		}
	}
	// Exec into the container to check if it's ready
	for {
		// Current version of Kubernetes Go client doesn't support "exec", so run this via the command line
		// TODO: If we run "chkmqready" here, it doesn't seem to work
		//out, _, err := runCommand(t, "kubectl", "exec", podName, "--", "dspmq")
		out, _, err := kubeExec(t, podName, "dspmq")
		//out, rc, err := runCommand(t, "kubectl", "exec", podName, "--", "chkmqready")
		if err != nil {
			t.Error(out)
			out2, _, err2 := runCommand(t, "kubectl", "describe", "pod", podName)
			if err2 == nil {
				t.Log(out2)
			}
			t.Fatal(err)
		}
		if strings.Contains(out, "Running") {
			t.Log("MQ is ready")
			return
		}
		// if rc == 0 {
		// 	return
		// }
		time.Sleep(1 * time.Second)
	}
}

func getPodsForHelmRelease(t *testing.T, cs *kubernetes.Clientset, release string) *v1.PodList {
	pods, err := cs.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: "release=" + release,
	})
	if err != nil {
		t.Fatal(err)
	}
	return pods
}

func storageClassesDefined(t *testing.T, cs *kubernetes.Clientset) bool {
	c, err := cs.Storage().StorageClasses().List(metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Items) > 0 {
		return true
	}
	return false
}

// volumesAvailable checks to see if any persistent volumes are available.
// On some Kubernetes clusters, only storage classes are used, so there won't
// be any volumes pre-created.
func volumesAvailable(t *testing.T, cs *kubernetes.Clientset) bool {
	pvs, err := cs.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, pv := range pvs.Items {
		if pv.Status.Phase == v1.VolumeAvailable {
			return true
		}
	}
	return false
}

// TODO: On Minikube, need to make sure Helm is initialized first
