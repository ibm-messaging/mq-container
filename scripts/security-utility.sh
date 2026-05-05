#!/bin/bash

# © Copyright IBM Corporation 2026
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This wrapper script safely executes the securityUtility command with user-provided
# arguments passed as parameters rather than being interpolated into a shell command.
# This approach eliminates shell injection vulnerabilities by ensuring user input
# is never interpreted by the shell.

set -e

# Source the MQ environment (sets up MQ_INSTALLATION_PATH, PATH, etc.)
source setmqenv -s || true

# Start the MQ web server component
/opt/mqm/web/bin/server || true

# Execute securityUtility with all arguments passed as parameters
# The "$@" ensures each argument is properly quoted and passed individually
exec /opt/mqm/web/bin/securityUtility "$@"

