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

import "github.com/ibm-messaging/mq-container/internal/ready"

// qmStatusLabel maps a QMStatus to the value for the mq.ibm.com/status label.
func qmStatusLabel(status ready.QMStatus) string {
	switch status {
	case ready.StatusActiveQM, ready.StatusRecoveryLeaderQM:
		return "active"
	case ready.StatusStandbyQM:
		return "standby"
	case ready.StatusReplicaQM:
		return "replica"
	case ready.StatusStarting:
		return "starting"
	default:
		return ""
	}
}
