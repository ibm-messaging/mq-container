# © Copyright IBM Corporation 2019
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

# Fedora includes more recent versions of buildah (need buildah V1.7 to get 
# multi-stage builds to work properly)
FROM docker.io/fedora:29
RUN yum install -y buildah
COPY build.sh /usr/local/bin/build
RUN chmod +x /usr/local/bin/build
ENV STORAGE_DRIVER=vfs
ENV BUILDAH_ISOLATION=chroot
ENTRYPOINT ["build"]
