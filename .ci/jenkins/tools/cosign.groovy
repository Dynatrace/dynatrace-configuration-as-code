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

cosignSecrets = [[path        : "keptn-jenkins/monaco/cosign",
                  secretValues: [
                      [envVar: 'cosign_key', vaultKey: 'cosign.key', isRequired: true],
                      [envVar: 'cosign_pub', vaultKey: 'cosign.pub', isRequired: true],
                      [envVar: 'cosign_password', vaultKey: 'cosign_password', isRequired: true]
                  ]]]

void install(String version) {
    sh(label: "install cosign tool", script: "go install github.com/sigstore/cosign/v2/cmd/cosign@${version}")
}

/**
 * sign is a wrapper function which calls cosing sign
 * @param image address in format registry/library/image:version (e.g. docker.io/dynatrace/dynatrace-configuration-as-code:2.17.0)
 */
void sign(String image) {
    withEnv(["image=${image}"]) {
        withVault(vaultSecrets: cosignSecrets) {
            sh(label: "sign container",
                script: 'COSIGN_PASSWORD=$cosign_password cosign sign --key=env://cosign_key $image -y --recursive')
        }
    }
}

String getPublicKey() {
    withVault(vaultSecrets: cosignSecrets) {
        return cosign_pub
    }
}

return this
