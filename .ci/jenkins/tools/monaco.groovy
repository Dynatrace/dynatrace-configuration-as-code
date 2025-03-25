/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

secrets = [[path        : "keptn-jenkins/monaco/integration-tests/performance",
            secretValues: [
                [envVar: 'OAUTH_CLIENT_ID', vaultKey: 'OAUTH_CLIENT_ID', isRequired: true],
                [envVar: 'OAUTH_CLIENT_SECRET', vaultKey: 'OAUTH_CLIENT_SECRET', isRequired: true],
                [envVar: 'OAUTH_TOKEN_ENDPOINT', vaultKey: 'OAUTH_TOKEN_ENDPOINT', isRequired: true],
                [envVar: 'TENANT_URL', vaultKey: 'TENANT_URL', isRequired: true],
                [envVar: 'TOKEN', vaultKey: 'TOKEN', isRequired: true],
                [envVar: 'LOG_FWD_URL', vaultKey: 'LOG_FWD_URL', isRequired: true],
                [envVar: 'LOG_FWD_TOKEN', vaultKey: 'LOG_FWD_TOKEN', isRequired: true]

            ]]]

void build(String sourcePath) {
    String monacoBin = "${JENKINS_AGENT_WORKDIR}/monaco"
    sh(label: "build monaco",
        script: """CGO_ENABLED=0
                go build
                  -a -tags=netgo -buildvcs=false
                  -ldflags=\"-X github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version.MonitoringAsCode=2.x -w -extldflags -static\"
                  -o ${monacoBin}
                  ${sourcePath}/cmd/monaco
            """.replaceAll("\n", " ").replaceAll(/ +/, " "))
}

void purge() {
    String monacoBin = "${JENKINS_AGENT_WORKDIR}/monaco"
    withVault(vaultSecrets: secrets) {
        sh(label: "purge tenant",
            script: "MONACO_ENABLE_DANGEROUS_COMMANDS=true ${monacoBin} purge manifest.yaml")
    }
}

void deploy(String project, boolean ignoreReturnStatus = true) {
    String monacoBin = "${JENKINS_AGENT_WORKDIR}/monaco"
    String logForwarderBin = "${JENKINS_AGENT_WORKDIR}/logForwarder"
    String manifestPath = "${JENKINS_AGENT_WORKDIR}/deployment/manifest.yaml"

    withVault(vaultSecrets: secrets) {
        // to provoke memory leak remove MONACO_CONCURENT_DEPLOYMENT flag. The default value is MONACO_CONCURENT_DEPLOYMENT=100
        status = sh(label: "monaco deploy",
            returnStatus: true,
            script: "MONACO_CONCURENT_DEPLOYMENT=30 MONACO_LOG_FORMAT=json ${monacoBin} deploy ${manifestPath} --project=${project} --verbose | ${logForwarderBin} LOG_FWD_URL LOG_FWD_TOKEN")
        if (!ignoreReturnStatus) {
            0 == status
        }
    }
}

void buildForwarder() {
    String logForwarderBin = "${JENKINS_AGENT_WORKDIR}/logForwarder"
   sh(label: "build monaco-log-forwarder",
        script: """CGO_ENABLED=0 go build -a -tags=netgo -buildvcs=false -ldflags=\"-w -extldflags -static\" -o ${logForwarderBin}""")
    sh "ls -alF ${JENKINS_AGENT_WORKDIR}"
}

void copyTestData() {
    String deploymentFolder = "${JENKINS_AGENT_WORKDIR}/deployment"
    sh(label: "copy test data",
        script: """cp -R . ${deploymentFolder}/""")
}

return this


