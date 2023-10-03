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

/**
 * installDockerQemuEmulator enables to run/build another architectures
 */
void installQemuEmulator() {
    sh "docker run --privileged --rm tonistiigi/binfmt --install all"
}

void createNewBuilder() {
    try { //in case of a quick rerun of the process, a container already has this builder
        sh "docker buildx use thebuilder"
    } catch (def e) {
        sh "docker buildx create --name thebuilder --bootstrap --use"
    }
}

void installCosign(){
    sh(label: "install cosign tool", script: "go install github.com/sigstore/cosign/v2/cmd/cosign@v2.1.1")
}

void listDrivers() {
    sh "docker buildx ls"
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
            try {
                sh(label: "sign in to docker registry", script: 'docker login --username $username --password $password $registry')

                String command = '''\
                    DOCKER_BUILDKIT=1
                    docker buildx build .
                      --progress=plain
                      --platform=arm64,amd64
                      --build-arg VERSION=$version
                      --push
                      --tag $registry/$repo/dynatrace-configuration-as-code:$version
                  '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
                if (args.latest) {
                    command = "${command} --tag $registry/$repo/dynatrace-configuration-as-code:latest"
                }
                sh(label: "build container", script: command)

                sh(label: "sing container",
                    script: 'COSIGN_PASSWORD=$cosign_password cosign sign --key env://cosign_key $registry/$repo/dynatrace-configuration-as-code:$version -y --recursive')
            } finally {
                sh(label: "sign out from docker registry", script: 'docker logout $registry')
            }
        }
    }
}

return this
