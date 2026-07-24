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
	"encoding/json"
	"fmt"
	"net/http"
)

// getRequiredPermissions returns the RBAC permissions this server needs in order to
// function correctly (patching pods).
func getRequiredPermissions(cfg *inClusterConfig) []requiredPermission {
	return []requiredPermission{
		{
			Verb:      "patch",
			Resource:  "pods",
			APIGroup:  "",
			Namespace: cfg.namespace,
			Name:      cfg.podName,
		},
	}
}

// checkPermission POSTs a single SelfSubjectAccessReview and returns an error
// if the server rejects the request or the service account is not allowed.
func checkPermission(ctx context.Context, cfg *inClusterConfig, url string, perm requiredPermission) error {
	payload := selfSubjectAccessReviewPayload{
		APIVersion: "authorization.k8s.io/v1",
		Kind:       "SelfSubjectAccessReview",
		Spec: selfSubjectAccessReviewSpec{
			ResourceAttributes: resourceAttributes{
				Verb:      perm.Verb,
				Resource:  perm.Resource,
				Group:     perm.APIGroup,
				Name:      perm.Name,
				Namespace: perm.Namespace,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling SelfSubjectAccessReview: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating SelfSubjectAccessReview request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cfg.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending SelfSubjectAccessReview: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status %s for permission %s %s", resp.Status, perm.Verb, perm.Resource)
	}

	var result selfSubjectAccessReviewStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding SelfSubjectAccessReview response: %w", err)
	}
	if !result.Status.Allowed {
		return missingPermissionsError{permission: perm, reason: result.Status.Reason}
	}
	return nil
}

// requiredPermission describes a single RBAC verb+resource that the service
// account must be allowed to perform. Name is optional and scopes the check
// to a specific named resource instance.
type requiredPermission struct {
	Verb      string
	Resource  string
	APIGroup  string
	Name      string
	Namespace string
}

func (r requiredPermission) String() string {
	return fmt.Sprintf("%q %s", r.Verb, r.scopedResource())
}

func (r requiredPermission) scopedResource() string {
	scope := r.scope()
	if scope == "" {
		return r.groupResource()
	}
	return fmt.Sprintf("%s[%s]", r.groupResource(), scope)
}

func (r requiredPermission) scope() string {
	name, namespace := r.Name, r.Namespace
	if namespace == "" && name == "" {
		return ""
	}
	if name == "" {
		name = "*"
	}
	if namespace == "" {
		namespace = "*"
	}
	return fmt.Sprintf("%s/%s", namespace, name)
}

func (r requiredPermission) groupResource() string {
	if r.APIGroup == "" {
		return r.Resource
	}
	return fmt.Sprintf("%s/%s", r.APIGroup, r.Resource)
}

type resourceAttributes struct {
	Verb      string `json:"verb"`
	Resource  string `json:"resource"`
	Group     string `json:"group"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type selfSubjectAccessReviewSpec struct {
	ResourceAttributes resourceAttributes `json:"resourceAttributes"`
}

type selfSubjectAccessReviewPayload struct {
	APIVersion string                      `json:"apiVersion"`
	Kind       string                      `json:"kind"`
	Spec       selfSubjectAccessReviewSpec `json:"spec"`
}

type selfSubjectAccessReviewStatus struct {
	Status struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
	} `json:"status"`
}

type missingPermissionsError struct {
	permission requiredPermission
	reason     string
}

func (m missingPermissionsError) Error() string {
	message := fmt.Sprintf("permission denied for %s", m.permission)

	if m.reason != "" {
		message += fmt.Sprintf(" (%s)", m.reason)
	}
	return message
}
