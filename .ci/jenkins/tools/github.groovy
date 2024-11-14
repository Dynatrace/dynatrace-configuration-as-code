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

githubSecrets = [[path        : 'keptn-jenkins/monaco/github-credentials',
                  secretValues: [[envVar: 'token', vaultKey: 'access_token', isRequired: true]]]]

int createRelease(Map args = [version: null]) {
    withEnv(["version=${args.version}",]) {
        withVault(vaultSecrets: githubSecrets) {
            def jsonRes = sh(returnStdout: true, label: "create release on GitHub", script: '''
                        curl --request POST https://api.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases
                             --header "Accept: application/vnd.github+json"
                             --header "Authorization: Bearer $token"
                             --header "X-GitHub-Api-Version: 2022-11-28"
                             --fail-with-body
                             --data \'{
                                        "tag_name":"v\'$version\'",
                                        "target_commitish":"main",
                                        "name":"\'$version\'",
                                        "draft":true,
                                        "prerelease":false,
                                        "generate_release_notes":true
                                    }\'
                        '''.replaceAll("\n", " ").replaceAll(/ +/, " "))
            return readJSON(text: jsonRes).id
        }
    }
}

void pushFileToRelease(Map args = [rleaseName: null, filename: null, releaseId: null]) {
    withEnv(["rleaseName=${args.rleaseName}", "filename=${args.filename}", "releaseId=${args.releaseId}",
    ]) {
        withVault(githubSecrets) {
            sh(label: "push file to GitHub", script: '''
                    curl --request POST https://uploads.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases/$releaseId/assets?name=$rleaseName
                         --header "Accept: application/vnd.github+json"
                         --header "Authorization: Bearer $token"
                         --header "X-GitHub-Api-Version: 2022-11-28"
                         --header "Content-Type: application/octet-stream"
                         --fail-with-body
                         --data-binary @"$filename"
                    '''.replaceAll("\n", " ").replaceAll(/ +/, " "))
        }
    }
}

void pushRawDataToRelease(Map args = [rleaseName: null, rawData: null, releaseId: null]) {
    withEnv(["rleaseName=${args.rleaseName}", "rawData=${args.rawData}", "releaseId=${args.releaseId}",
    ]) {
        withVault(githubSecrets) {
            sh(label: "push raw data to GitHub", script: '''
                    curl --request POST https://uploads.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases/$releaseId/assets?name=$rleaseName
                         --header "Accept: application/vnd.github+json"
                         --header "Authorization: Bearer $token"
                         --header "X-GitHub-Api-Version: 2022-11-28"
                         --header "Content-Type: application/octet-stream"
                         --fail-with-body
                         --data-raw "$rawData"
                    '''.replaceAll("\n", " ").replaceAll(/ +/, " "))
        }
    }
}


return this
