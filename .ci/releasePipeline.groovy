final String PROJECT = 'monaco'
def version
def releaseId

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

                            version = getVersionFromGitTagName()
                            releaseId = createGitHubRelease(version: version)

                            echo "version= ${version}"
                            echo "GitHub releaseId= ${releaseId}"
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

                                                buildBinary(command: release, version: version, dest: "${release}")
                                                signWinBinaries(source: release, version: version, destDir: '.', projectName: PROJECT)
                                                pushToDynatraceStorage(source: release, version: version)
                                                pushToGithub(source: release, releaseId: releaseId)
                                                break
                                            case "linux":
                                            case "darwin":
                                                def release = "${PROJECT}-${env.OS}-${env.ARCH}" //name of the tar archive

                                                sh "rm -rf ${release}" //ensure to delete old build

                                                buildBinary(command: release, version: version, dest: "${release}/${PROJECT}")
                                                createShaSum(source: "${release}/${PROJECT}", dest: "${release}/${PROJECT}.p7m")
                                                signShaSum(source: "${release}/${PROJECT}.p7m", version: version, destDir: release, projectName: PROJECT)
                                                createTarArchive(source: release, dest: release + '.tar')
                                                pushToDynatraceStorage(source: release + '.tar', version: version)
                                                pushToGithub(source: release + '.tar', releaseId: releaseId)
                                                break
                                            case "container":
                                                createContainerAndPushToStorage(version: version)
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
        signWithSignService(source: args.source, version: args.version, destDir: args.destDir, signAction: "SIGN", projectName: args.projectName)
    }
}

void createShaSum(Map args = [source: null, dest: null]) {
    stage('Compute SHA sum') {
        sh "sha256sum ${args.source} | sed 's, .*/, ,' > ${args.dest}"
    }
}

void signShaSum(Map args = [source: null, version: null, destDir: null, projectName: null]) {
    stage('Sign SHA sum') {
        signWithSignService(source: args.source, version: args.version, destDir: args.destDir, signAction: "ENVELOPE", projectName: args.projectName)
        sh "cat ${args.source}"
    }
}

void createTarArchive(Map args = [source: null, dest: null]) {
    stage('Create tar archive') {
        tar(dir: args.source, file: args.dest)
    }
}

def pushToDynatraceStorage(Map args = [source: null, version: null]) {
    stage('Deliver tar to storage') {
        withEnv(["source=${args.source}", "version=${args.version}",]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/artifact-storage-deploy',
                                      secretValues: [
                                          [envVar: 'repo_path', vaultKey: 'repo_path', isRequired: true],
                                          [envVar: 'username', vaultKey: 'username', isRequired: true],
                                          [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
            ) {
                sh '''
                    curl --request PUT $repo_path/monaco/$version/$source
                         --user "$username":"$password"
                         --fail-with-body
                         --upload-file $source
                   '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
            }
        }
    }
}

void signWithSignService(Map args = [source: null, version: null, destDir: null, signAction: null, projectName: null]) {
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

void createContainerAndPushToStorage(Map args = [version: null]) {
    stage('📦 Release Container Image') {
        withEnv(["version=${args.version}"]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/registry-deploy',
                                      secretValues: [[envVar: 'registry', vaultKey: 'registry_path', isRequired: true],
                                                     [envVar: 'username', vaultKey: 'username', isRequired: true],
                                                     [envVar: 'password', vaultKey: 'password', isRequired: true]]]]) {
                script {
                    try {
                        sh 'docker login --username $username --password $password $registry'
                        sh 'DOCKER_BUILDKIT=1 make docker-container OUTPUT=./build/docker/monaco CONTAINER_NAME=$registry/monaco/dynatrace-monitoring-as-code VERSION=$version'
                        sh 'docker push $registry/monaco/dynatrace-monitoring-as-code:$version'
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

void pushToGithub(Map args = [source: '', releaseId: '']) {
    stage('Deliver to GitHub') {
        withEnv(["source=${args.source}", "releaseId=${args.releaseId}",
        ]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/github-credentials',
                                      secretValues: [[envVar: 'token', vaultKey: 'access_token', isRequired: true]]]]
            ) {
                sh '''
                    curl --request POST https://uploads.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases/$releaseId/assets?name=$source
                         --header "Accept: application/vnd.github+json"
                         --header "Authorization: Bearer $token"
                         --header "X-GitHub-Api-Version: 2022-11-28"
                         --header "Content-Type: application/octet-stream"
                         --fail-with-body
                         --data-binary @$source
                    '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
            }
        }
    }
}
