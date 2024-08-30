/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

 void installKo() {
  sh(label: "install ko build-tool", script: "make install-ko")
 }

void installCosign() {
    sh(label: "install cosign signature-tool", script: "go install github.com/sigstore/cosign/v2/cmd/cosign@v2.1.1")
}

void buildContainer(Map args = [version: null, registrySecretsPath: null, latest: false]) {
    withEnv(["version=${args.version}"]) {
        withVault(vaultSecrets: [
            [
                path        : "${args.registrySecretsPath}",
                secretValues: [
                    [envVar: 'registry', vaultKey: 'registry', isRequired: true],
                    [envVar: 'repo', vaultKey: 'repo', isRequired: true],
                    [envVar: 'username', vaultKey: 'username', isRequired: true],
                    [envVar: 'password', vaultKey: 'password', isRequired: true]
                ]
            ],
            [
                path        : "keptn-jenkins/monaco/cosign",
                secretValues: [
                    [envVar: 'cosign_key', vaultKey: 'cosign.key', isRequired: true],
                    [envVar: 'cosign_pub', vaultKey: 'cosign.pub', isRequired: true],
                    [envVar: 'cosign_password', vaultKey: 'cosign_password', isRequired: true]
                ]
            ]
        ]) {
            sh(label: "sign in to container registry",
                script: 'ko login --username $username --password $password $registry')

            String tags = "$version"
            if (args.latest) {
                tags += ",latest"
            }
            sh(label: "build and publish container",
                script: "make docker-container REPO_PATH=$registry/$repo VERSION=$version TAGS=${tags}")

            sh(label: "sign container",
                script: 'COSIGN_PASSWORD=$cosign_password cosign sign --key env://cosign_key $registry/$repo/dynatrace-configuration-as-code:$version -y --recursive')
        }
    }
}

return this
