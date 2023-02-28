import groovy.transform.Field

@Field Map ctx = [unsignedDir: "./unsigned",
                  signedDir  : "./signed"]

def version
def releaseID

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
        stage('process tags') {
            when {
                tag 'v*'
            }
            stages {
                stage('Prepare for build') {
                    steps {
                        script {
                            version = getVersionFromGitTagName()
                            getSignClient()
                            sh "mkdir -p ${ctx.unsignedDir}"
                            sh "mkdir -p ${ctx.signedDir}"
                        }
                    }
                }
                stage('Create GitHub release draft') {
                    steps {
                        script {
                            withEnv(["VERSION=${version}",]) {
                                withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/github-credentials',
                                                          secretValues: [
                                                              [envVar: 'token', vaultKey: 'access_token', isRequired: true]
                                                          ]]]
                                ) {
                                    releaseID = sh(returnStdout: true, script: '''curl \
                                      -X POST \
                                      -H "Accept: application/vnd.github+json" \
                                      -H "Authorization: Bearer $TOKEN"\
                                      -H "X-GitHub-Api-Version: 2022-11-28" \
                                      https://api.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases \
                                      -d "{\"tag_name\":\"v$VERSION\",\"target_commitish\":\"main\",\"name\":\"$VERSION\",\"draft\":true,\"prerelease\":false,\"generate_release_notes\":true}" \
                                      | jq -r ".id" ''')
                                }
                            }
                        }
                    }
                }
                stage('matrix') {
                    matrix {
                        axes {
                            axis {
                                name 'PLATFORM'
                                values 'monaco-windows-386.exe', 'monaco-windows-amd64.exe'
                            }
                        }
                        stages {
                            stage('build') {
                                steps {
                                    buildBinary(binary: env.PLATFORM, version: version)
                                    signBinaries(binary: env.PLATFORM, version: version)
                                    releaseBinaryToArtifactory(binary: env.PLATFORM, version: version)
                                    releaseBinaryToGithub(binary: env.PLATFORM, version: version, releaseID: releaseID)
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
    echo "version= ${ver}"
    return ver
}

void getSignClient() {
    withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/sign-service',
                              secretValues: [[envVar: 'repository', vaultKey: 'client_repository_url ', isRequired: true]]]]
    ) {
        if (!fileExists('sign-client.jar')) {
            sh 'curl -Ss -g -o sign-client.jar $repository/testautomation-release-local/com/dynatrace/services/signservice/client/1.0.[RELEASE]/client-1.0.[RELEASE].jar'
        }
    }
}

void buildBinary(Map args = [binary: '', version: '', ctx: null]) {
    args.ctx = args.ctx ?: ctx
    stage('🏁 Build ' + args.binary) {
        sh "make ${args.binary} VERSION=${args.version} OUTDIR=${args.ctx.unsignedDir}"
    }
}

void signBinaries(Map args = [binary: '', version: '', ctx: null]) {
    args.ctx = args.ctx ?: ctx
    def os = args.binary.split('-')
    echo "${os}"
    echo "${os[1]}"
    stage('Sign ' + args.binary) {
        script {
            switch (args.binary.split('-')[1]) {
                case "windows":
                    signWinBinaries(args)
                    break
            }
        }
    }
}

void signWinBinaries(Map args = [binary: '', version: '', ctx: null]) {
    withEnv(["binarySource=${args.ctx.unsignedDir}/${args.binary}", "version=${args.version}", "destinationDir=${args.ctx.signedDir}"]) {
        withVault(vaultSecrets: [[path        : 'signing-service-authentication/monaco',
                                  secretValues: [
                                      [envVar: 'username', vaultKey: 'username', isRequired: true],
                                      [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
        ) {
            sh '''java -jar sign-client.jar \
                        --action SIGN \
                        --digestAlg SHA256 \
                        --downloadPath $destinationDir \
                        --kernelMode false  \
                        --triggeringProject datasource-go \
                        --triggeringVersion $version \
                        --username $username \
                        --password $password \
                        $binarySource'''
        }
    }
}



def releaseBinaryToArtifactory(Map args = [binary: '', version: '', ctx: null]) {
    args.ctx = args.ctx ?: ctx
    stage('Deliver ' + args.binary + ' internally') {
        withEnv(["binary=${args.binary}", "version=${args.version}", "signedDir=${args.ctx.signedDir}",]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/artifact-storage-deploy',
                                      secretValues: [
                                          [envVar: 'repo_path', vaultKey: 'repo_path', isRequired: true],
                                          [envVar: 'username', vaultKey: 'username', isRequired: true],
                                          [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
            ) {
                sh 'curl -u "$username":"$password" -X PUT $repo_path/monaco/$version/$binary -T $signedDir/$binary'
            }
        }
    }
}

def releaseBinaryToGithub(Map args = [binary: '', version: '', releaseID: '', ctx: null]) {
    args.ctx = args.ctx ?: ctx
    stage('Deliver ' + args.binary + ' to GitHub') {
        withEnv(["binary=${args.binary}", "version=${args.version}", "signedDir=${args.ctx.signedDir}", "releaseID=${args.releaseID}",]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/github-credentials',
                                      secretValues: [
                                          [envVar: 'token', vaultKey: 'access_token', isRequired: true]
                                      ]]]
            ) {
                sh '''curl -X POST -H "Accept: application/vnd.github+json" \
                -H "Authorization: Bearer $token"\
                -H "X-GitHub-Api-Version: 2022-11-28" \
                -H "Content-Type: application/octet-stream" \
                https://uploads.github.com/repos/Dynatrace/dynatrace-configuration-as-code/releases/$releaseID/assets?name=$binary \
                --data-binary "$signedDir/$binary"'''
            }
        }
    }
}

void releaseContainerToArtifactory() {
    def registrySecrets = [
        path        : 'keptn-jenkins/monaco/registry-deploy',
        secretValues: [
            [envVar: 'REGISTRY_PATH', vaultKey: 'registry_path', isRequired: true],
            [envVar: 'REGISTRY_USERNAME', vaultKey: 'username', isRequired: true],
            [envVar: 'REGISTRY_PASSWORD', vaultKey: 'password', isRequired: true]
        ]
    ]

    stage('📦 Release Container Image') {
        steps {
            withEnv(["VERSION=${VERSION}"]) {
                withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/registry-deploy',
                                          secretValues: [
                                              [envVar: 'REGISTRY_PATH', vaultKey: 'registry_path', isRequired: true],
                                              [envVar: 'REGISTRY_USERNAME', vaultKey: 'username', isRequired: true],
                                              [envVar: 'REGISTRY_PASSWORD', vaultKey: 'password', isRequired: true]]]]
                ) {
                    script {
                        sh 'docker login $REGISTRY_PATH -u "$REGISTRY_USERNAME" -p "$REGISTRY_PASSWORD" '
                        sh 'docker build -t $REGISTRY_PATH/monaco/dynatrace-monitoring-as-code:$VERSION .'
                        sh 'docker push $REGISTRY_PATH/monaco/dynatrace-monitoring-as-code:$VERSION'
                    }
                }
            }
        }
        post {
            always {
                withVault(vaultSecrets: [registrySecrets]) {
                    sh 'docker logout $REGISTRY_PATH'
                }
            }
        }
    }

}
