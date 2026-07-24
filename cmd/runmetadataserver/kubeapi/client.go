/*
© Copyright IBM Corporation 2026

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

package kubeapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-container/internal/ready"
)

type Client struct {
	loadConfig func() (*inClusterConfig, error)
	meta       *podMetaReader
}

func NewClient() *Client {
	return &Client{
		loadConfig: loadInClusterConfig,
		meta:       newPodMetaReader(),
	}
}

func (k *Client) VerifyPermissions(ctx context.Context) error {
	cfg, err := k.loadConfig()
	if err != nil {
		return fmt.Errorf("loading in-cluster config: %w", err)
	}
	permissions := getRequiredPermissions(cfg)
	return k.selfSubjectAccessReview(ctx, cfg, permissions)
}

func (k *Client) selfSubjectAccessReview(ctx context.Context, cfg *inClusterConfig, permissions []requiredPermission) error {
	url := cfg.host + "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews"
	missingPermissionStrings := []string{}

	for _, perm := range permissions {
		if err := checkPermission(ctx, cfg, url, perm); err != nil {
			if permErr, isPermErr := errors.AsType[missingPermissionsError](err); isPermErr {
				missingPermissionStrings = append(missingPermissionStrings, permErr.Error())
			} else {
				return err
			}
		}
	}
	if len(missingPermissionStrings) > 0 {
		return fmt.Errorf("missing required permissions: %s", strings.Join(missingPermissionStrings, ", "))
	}
	return nil
}

func (k *Client) ValidateStatusAnnotations() error {
	annotations, err := k.meta.annotations()
	if err != nil {
		return err
	}
	if _, found := annotations[annotationWhenActive]; !found {
		return nil
	}

	haveReplicaAnnotations := false
	if replicaAnnotations, found := annotations[annotationWhenReplica]; found {
		haveReplicaAnnotations = true
		err = checkJsonKeysMatch(annotations[annotationWhenActive], replicaAnnotations)
		if err != nil {
			return err
		}
	}
	haveStandbyAnnotations := false
	if standbyAnnotations, found := annotations[annotationWhenStandby]; found {
		haveStandbyAnnotations = true
		err = checkJsonKeysMatch(annotations[annotationWhenActive], standbyAnnotations)
		if err != nil {
			return err
		}
	}
	if haveReplicaAnnotations && haveStandbyAnnotations {
		return errors.New("both standby and replica annotations defined")
	}
	return nil
}

func checkJsonKeysMatch(a, b string) error {
	var err error

	setA, setB := map[string]string{}, map[string]string{}

	err = json.Unmarshal([]byte(a), &setA)
	if err != nil {
		return fmt.Errorf("failed to parse annotation: %w", err)
	}
	err = json.Unmarshal([]byte(b), &setB)
	if err != nil {
		return fmt.Errorf("failed to parse annotation: %w", err)
	}

	for keyA := range setA {
		if _, found := setB[keyA]; !found {
			return fmt.Errorf("active/non-active keys do not match (missing %s)", keyA)
		}
	}
	for keyB := range setB {
		if _, found := setA[keyB]; !found {
			return fmt.Errorf("active/non-active keys do not match (missing %s)", keyB)
		}
	}
	return nil
}

// PatchPodMetadataIfNeeded issues a strategic-merge-patch updating the
// mq.ibm.com/status label and the license annotations on this pod.
// It is a no-op when the downward API files show the pod metadata already
// matches the desired state.
func (k *Client) PatchPodMetadataIfNeeded(ctx context.Context, status ready.QMStatus) error {
	labelValue := qmStatusLabel(status)
	if labelValue == "" {
		return fmt.Errorf("no label value for status %v", status)
	}

	licenseAnnotations, err := k.meta.resolveLicenseAnnotations(status)
	if err != nil {
		return fmt.Errorf("resolving license annotations: %w", err)
	}

	if k.meta.currentMatchesDesired(labelValue, licenseAnnotations) {
		return nil
	}

	p := podPatch{
		Metadata: podPatchMetadata{
			Labels:      map[string]string{labelMQStatus: labelValue},
			Annotations: licenseAnnotations,
		},
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshaling patch body: %w", err)
	}

	return k.patch(ctx, body)
}

// patch sends a strategic-merge-patch with the given body to the pod resource.
// It reloads config on every call so that rotated tokens are always picked up.
func (k *Client) patch(ctx context.Context, body []byte) error {
	cfg, err := k.loadConfig()
	if err != nil {
		return fmt.Errorf("loading in-cluster config: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s",
		cfg.host, cfg.namespace, cfg.podName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating PATCH request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.token)
	req.Header.Set("Content-Type", "application/strategic-merge-patch+json")

	resp, err := cfg.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pod PATCH returned %s", resp.Status)
	}
	return nil
}

// podPatchMetadata is the metadata section of a Kubernetes strategic-merge-patch body.
type podPatchMetadata struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// podPatch is the top-level body for a Kubernetes strategic-merge-patch on a pod.
type podPatch struct {
	Metadata podPatchMetadata `json:"metadata"`
}

func loadInClusterConfig() (*inClusterConfig, error) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("KUBERNETES_SERVICE_HOST or KUBERNETES_SERVICE_PORT not set")
	}

	tokenBytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("reading service account token: %w", err)
	}

	nsBytes, err := os.ReadFile(namespaceFile)
	if err != nil {
		return nil, fmt.Errorf("reading namespace: %w", err)
	}

	caBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("reading CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		return nil, fmt.Errorf("POD_NAME env var not set")
	}

	return &inClusterConfig{
		host:      fmt.Sprintf("https://%s:%s", host, port),
		token:     string(tokenBytes),
		namespace: string(nsBytes),
		podName:   podName,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: caPool},
			},
		},
	}, nil
}

// serviceAccountDir is the mount path for the projected service-account volume.
const serviceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"

var (
	tokenFile     = serviceAccountDir + "/token"
	caFile        = serviceAccountDir + "/ca.crt"
	namespaceFile = serviceAccountDir + "/namespace"
)

// inClusterConfig holds the resolved values needed to make Kubernetes API calls.
type inClusterConfig struct {
	host      string
	token     string
	namespace string
	podName   string
	client    httpDoer
}

// httpDoer is the subset of *http.Client used by the Kubernetes helpers.
// It is satisfied by *http.Client and can be replaced by a fake in tests.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}
