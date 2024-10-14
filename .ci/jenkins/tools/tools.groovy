/*
 * @license
 * Copyright 2024 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

void installGo(String version) {
    sh(label: "", script: "set -eux && export DEBIAN_FRONTEND=noninteractive")
    sh(label: "install necessary tools", script: "apt-get update &&  apt-get install -y --no-install-recommends g++ gcc libc6-dev make pkg-config")
    sh(label: "remove old go version", script: "rm -rf /usr/local/go")
    sh(label: "download go binaries", script: "curl --silent --show-error --location \"https://go.dev/dl/go${version}.linux-amd64.tar.gz\" | tar -xz -C /usr/local")
    sh(label: "set PATH", script: 'GOROOT=$(/usr/local/go/bin/go env GOROOT) && GOPATH=$(/usr/local/go/bin/go env GOPATH) && PATH="$PATH:$GOROOT/bin:$GOPATH/bin"')
    sh(label: "install gotestsum", script: "go install gotest.tools/gotestsum@latest")
}

return this
