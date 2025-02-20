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

void install() {
    sh(label: "install ko build-tool", script: "make install-ko")
}

private def registrySecret(String registry) {
    String path
    switch (registry) {
        case "DockerHub": path = "keptn-jenkins/monaco/dockerhub-deploy"; break
        case "DT": path = "keptn-jenkins/monaco/registry-deploy"; break
        default: path = "keptn-jenkins/monaco/registry-deploy"; break
    }

    return [[path        : "${path}",
             secretValues: [
                 [envVar: 'registry',        vaultKey: 'registry',        isRequired: true],
                 [envVar: 'repo',            vaultKey: 'repo',            isRequired: true],
                 [envVar: 'username',        vaultKey: 'username',        isRequired: true],
                 [envVar: 'password',        vaultKey: 'password',        isRequired: true],
                 [envVar: 'base_image_path', vaultKey: 'base_image_path', isRequired: true]
             ]
         ]]
}


void loginToRegistry(Map args = [registry: null]) {
    withVault(vaultSecrets: registrySecret(args.registry)) {
        sh(label: "sign in to ${args.registry} registry",
            script: 'ko login --username=$username --password=$password $registry')
    }
}

/**
 * @return image address in format registry/library/image:version (e.g. docker.io/dynatrace/dynatrace-configuration-as-code:2.17.0)
 */
String buildContainer(Map args = [tags: null, registry: null]) {
    withEnv(["tags=${args.tags.join(",")}", "version=${args.tags[0]}"]) { //, "vaultPath=${args.registrySecretsPath}"
        withVault(vaultSecrets: registrySecret(args.registry)) {
            sh(label: "build and publish container",
                script: 'make docker-container REPO_PATH=$registry/$repo VERSION=$version TAGS=$tags KO_BASE_IMAGE_PATH=$base_image_path' )
            return "$registry/$repo/dynatrace-configuration-as-code:$version"
        }
    }
}

return this
