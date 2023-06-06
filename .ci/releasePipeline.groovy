def fullRegex = ~/^\d+\.\d+\.\d+$/         // 1.2.3
def rcRegex = ~/^\d+\.\d+\.\d+-rc.*$/      // 1.2.3-rc(.*)
def devRegex = ~/^\d+\.\d+\.\d+-dev.*$/    // 1.2.3-dev(.*)

final String PROJECT = 'monaco'
def version
def releaseId

def isFullRelease, isRcRelease, isDevRelease

pipeline {
    agent {
        kubernetes {
            label 'ca-jenkins-agent'
            cloud 'linux-amd64'
            namespace 'keptn-jenkins-slaves-ni'
            nodeSelector 'beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux'
            instanceCap '2'
            idleMinutes '2'
            yamlFile '.ci/jenkins_agents/ca-jenkins-agent.yaml'
        }
    }

    stages {
        stage('🚀 Release') {
            when {
                tag 'v*'
            }
            stages {
                stage('Prepare for build') {
                    steps {
                        script {
                            getSignClient()

                            // version is without the v* prefix
                            version = getVersionFromGitTagName()

                            isFullRelease = fullRegex.matcher(version).matches()
                            isRcRelease = rcRegex.matcher(version).matches()
                            isDevRelease = devRegex.matcher(version).matches()

                            echo "version= ${version}"
                            echo "isFullRelease=${isFullRelease}"
                            echo "isRcRelease=${isRcRelease}"
                            echo "isDevRelease=${isDevRelease}"

                            if (!isFullRelease && !isRcRelease && !isDevRelease) {
                                error('Given version tag is not a valid tag. Use v1.2.3, v1.2.3-rc*, or v1.2.3-dev*')
                            }

                            if (!isDevRelease) {
                                releaseId = createGitHubRelease(version: version)
                                echo "GitHub releaseId=${releaseId}"
                            } else {
                                echo "Skipping creation of GitHub release for version ${version}"
                            }
                        }
                    }
                }

                stage('matrix') {
                    matrix {
                        axes {
                            axis {
                                name 'OS'
                                values 'windows', 'linux', 'darwin', 'container'
                            }
                            axis {
                                name 'ARCH'
                                values '386', 'amd64', 'arm64'
                            }
                        }
                        excludes {
                            exclude {
                                axis {
                                    name 'OS'
                                    values 'windows'
                                }
                                axis {
                                    name 'ARCH'
                                    values 'arm64'
                                }
                            }
                            exclude {
                                axis {
                                    name 'OS'
                                    values 'darwin'
                                }
                                axis {
                                    name 'ARCH'
                                    values '386'
                                }
                            }
                            exclude {
                                axis {
                                    name 'OS'
                                    values 'container'
                                }
                                axis {
                                    name 'ARCH'
                                    values '386', 'arm64'
                                }
                            }
                        }
                        stages {
                            stage('build') {
                                steps {
                                    script {
                                        switch (env.OS) {
                                            case "windows":
                                                def release = "${PROJECT}-${env.OS}-${env.ARCH}.exe"

                                                sh "rm -rf ${release}" //ensure to delete old build

                                                buildBinary(command: release, version: version, dest: release)
                                                signWinBinaries(source: release, version: version, destDir: '.', projectName: PROJECT)
                                                pushToDynatraceStorage(source: release, dest: "${PROJECT}/${version}/${release}")
                                                if (!isDevRelease) {
                                                    pushFileToGithub(rleaseName: release, source: release, releaseId: releaseId)
                                                } else {
                                                    echo "Skipping upload of binaries to GitHub"
                                                }
                                                break
                                            case "linux":
                                            case "darwin":
                                                def release = "${PROJECT}-${env.OS}-${env.ARCH}"

                                                sh "rm -rf ${release}" //ensure to delete old build

                                                buildBinary(command: release, version: version, dest: "${release}/${release}")
                                                createShaSum(source: "${release}/${release}", dest: "${release}/${release}.sha256")
//                                                signShaSum(source: "${release}/${release}.p7m", version: version, destDir: "${release}", projectName: PROJECT)
//                                                createTarArchive(source: release, dest: release + '.tar')

                                                def pathInStorage = "${PROJECT}/${version}/$release"
                                                pushToDynatraceStorage(source: "${release}/${release}", dest: pathInStorage)
                                                pushToDynatraceStorage(source: "${release}/${release}.sha256", dest: pathInStorage + ".sha256")
                                                if (!isDevRelease) {
                                                    pushFileToGithub(rleaseName: release, source: "${release}/${release}", releaseId: releaseId)
                                                    pushFileToGithub(rleaseName: release + ".sha256", source: "${release}/${release}.sha256", releaseId: releaseId)
                                                } else {
                                                    echo "Skipping upload of binaries to GitHub"
                                                }
                                                break
                                            case "container":
                                                // docker hub, only full releases and rc-releases
                                                if (!isDevRelease) {
                                                    createContainerAndPushToStorage(version: version, tagLatest: true, registrySecretsPath: 'keptn-jenkins/monaco/dockerhub-deploy', registry: 'Docker')
                                                } else {
                                                    echo "Skipping release of ${version} to Docker Hub"
                                                }

                                                // internal registry, all release are published
                                                tagLatest = !isDevRelease // only tag full-releases and rc-releases as 'latest'
                                                createContainerAndPushToStorage(version: version, tagLatest: tagLatest, registrySecretsPath: 'keptn-jenkins/monaco/registry-deploy', registry: 'Internal')

                                                if (!isDevRelease) {
                                                    withVault(vaultSecrets:
                                                        [[
                                                             path        : "keptn-jenkins/monaco/cosign",
                                                             secretValues: [[envVar: 'cosign_pub', vaultKey: 'cosign.pub', isRequired: true],]
                                                         ]]) {
                                                        pushToGithub(releaseId: releaseId, rleaseName: "cosign.pub", source: "$cosign_pub")
                                                    }
                                                }

                                                break
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    post {
        failure {
            emailext recipientProviders: [culprits()], subject: '$DEFAULT_SUBJECT', mimeType: 'text/html', body: '$DEFAULT_CONTENT'
        }
        unstable {
            emailext recipientProviders: [culprits()], subject: '$DEFAULT_SUBJECT', mimeType: 'text/html', body: '$DEFAULT_CONTENT'
        }
    }
}


String getVersionFromGitTagName() {
    def ver = sh(returnStdout: true, script: "git tag -l --points-at HEAD --sort=-creatordate | head -n 1")
        .trim()
        .substring(1)  // drop v prefix
    return ver
}

void getSignClient() {
    withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/sign-service',
                              secretValues: [[envVar: 'repository', vaultKey: 'client_repository_url ', isRequired: true]]]]
    ) {
        if (!fileExists('sign-client.jar')) {
            sh '''
                curl --request GET $repository/testautomation-release-local/com/dynatrace/services/signservice/client/1.0.[RELEASE]/client-1.0.[RELEASE].jar
                     --show-error
                     --silent
                     --globoff
                     --fail-with-body
                     --output sign-client.jar
               '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
        }
    }
}

void buildBinary(Map args = [command: null, version: null, dest: null]) {
    stage('🏁 Build ' + args.command) {
        sh "make ${args.command} VERSION=${args.version} OUTPUT=${args.dest}"
    }
}

void signWinBinaries(Map args = [source: null, version: null, destDir: null, projectName: null]) {
    stage('Sign binaries') {
        signWithSignService(source: args.source, version: args.version, destDir: args.destDir, projectName: args.projectName, signAction: "SIGN")
    }
}

void createShaSum(Map args = [source: null, dest: null]) {
    stage('Compute SHA sum') {
        sh "sha256sum ${args.source} | sed 's, .*/,  ,' > ${args.dest}"
    }
}

void signShaSum(Map args = [source: null, version: null, destDir: null, projectName: null]) {
    stage('Sign SHA sum') {
        signWithSignService(source: args.source, version: args.version, destDir: args.destDir, projectName: args.projectName, signAction: "ENVELOPE")
        sh "cat -v ${args.source}"
    }
}

void createTarArchive(Map args = [source: null, dest: null]) {
    stage('Create tar archive') {
        tar(dir: args.source, file: args.dest)
    }
}

def pushToDynatraceStorage(Map args = [source: null, dest: null]) {
    stage('Deliver to storage') {
        withEnv(["source=${args.source}", "dest=$args.dest"]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/artifact-storage-deploy',
                                      secretValues: [
                                          [envVar: 'repo_path', vaultKey: 'repo_path', isRequired: true],
                                          [envVar: 'username', vaultKey: 'username', isRequired: true],
                                          [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
            ) {
                sh '''
                    curl --request PUT $repo_path/$dest
                         --user "$username":"$password"
                         --fail-with-body
                         --upload-file $source
                   '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
            }
        }
    }
}

void signWithSignService(Map args = [source: null, version: null, destDir: '.', signAction: null, projectName: null]) {
    withEnv(["source=${args.source}", "version=${args.version}", "destDir=${args.destDir}", "signAction=${args.signAction}", "project=${args.projectName}"]) {
        withVault(vaultSecrets: [[path        : 'signing-service-authentication/monaco',
                                  secretValues: [
                                      [envVar: 'username', vaultKey: 'username', isRequired: true],
                                      [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
        ) {
            sh '''
                java -jar sign-client.jar
                  --action $signAction
                  --digestAlg SHA256
                  --downloadPath $destDir
                  --kernelMode false
                  --triggeringProject $project
                  --triggeringVersion $version
                  --username $username
                  --password $password
                  $source
               '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
        }
    }

}

void createContainerAndPushToStorage(Map args = [version: null, tagLatest: false, registrySecretsPath: null, registry: null]) {
    stage("Publish container: registry=${args.registry}, version=${args.version}") {
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
                        [envVar: 'cosign_password', vaultKey: 'cosign_password', isRequired: true]
                    ]
                ]
            ]) {
                script {
                    try {
                        sh 'docker login --username $username --password $password $registry'
                        sh 'DOCKER_BUILDKIT=1 make docker-container OUTPUT=./build/docker/monaco CONTAINER_NAME=$registry/$repo/dynatrace-configuration-as-code VERSION=$version'

                        sh 'docker push $registry/$repo/dynatrace-configuration-as-code:$version'

                        sh 'make sign-image COSIGN_PASSWORD=$cosign_password FULL_IMAGE_NAME=$registry/$repo/dynatrace-configuration-as-code:$version'

                        if (args.tagLatest) {
                            sh 'docker tag $registry/$repo/dynatrace-configuration-as-code:$version $registry/$repo/dynatrace-configuration-as-code:latest'
                            sh 'docker push $registry/$repo/dynatrace-configuration-as-code:latest'
                        }
                    } finally {
                        sh 'docker logout $registry'
                    }
                }
            }
        }
    }
}

int createGitHubRelease(Map args = [version: null]) {
    stage('Create GitHub release draft') {
        script {
            withEnv(["version=${args.version}",]) {
                withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/github-credentials',
                                          secretValues: [[envVar: 'token', vaultKey: 'access_token', isRequired: true]]]]
                ) {
                    def jsonRes = sh(returnStdout: true, script: '''
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
                    def jsonResObj = readJSON(text: jsonRes)
                    return jsonResObj.id
                }
            }
        }
    }
}

void pushToGithub(Map args = [rleaseName: null, source: null, releaseId: null]) {
    stage('Deliver to GitHub') {
        script {
            withEnv(["rleaseName=${args.rleaseName}", "source=${args.source}", "releaseId=${args.releaseId}",
            ]) {
                withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/github-credentials',
                                          secretValues: [[envVar: 'token', vaultKey: 'access_token', isRequired: true]]]]
                ) {
                    sh '''
                    curl --request POST https://uploads.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases/$releaseId/assets?name=$rleaseName
                         --header "Accept: application/vnd.github+json"
                         --header "Authorization: Bearer $token"
                         --header "X-GitHub-Api-Version: 2022-11-28"
                         --header "Content-Type: application/octet-stream"
                         --fail-with-body
                         --data-binary "$source"
                    '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
                }
            }
        }
    }
}


void pushFileToGithub(Map args = [rleaseName: null, source: null, releaseId: null]) {
    pushToGithub(releaseId: args.releaseId, source: "@" + args.source, rleaseName: args.rleaseName)
}
