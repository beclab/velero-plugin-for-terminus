# Copyright 2017, 2019 the Velero contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.22 as builder

WORKDIR /workspace

# Copy the go source
COPY . ./

# Build
RUN CGO_ENABLED=0 go build -a -o ./_output/velero-plugin-for-terminus ./velero-plugin-for-terminus/ && CGO_ENABLED=0 go build -a -o ./_output/cp-plugin ./hack/cp-plugin

FROM gcr.io/distroless/static:debug
WORKDIR /
COPY --from=builder /workspace/_output/velero-plugin-for-terminus /plugins/
COPY --from=builder /workspace/_output/cp-plugin /bin/cp-plugin
USER 65532:65532
ENTRYPOINT ["cp-plugin", "/plugins/velero-plugin-for-terminus", "/target/velero-plugin-for-terminus"]
