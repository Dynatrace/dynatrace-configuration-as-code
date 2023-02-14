def artifactStorageSecrets = [
    path        : 'keptn-jenkins/monaco/artifact-storage-deploy',
    secretValues: [
        [envVar: 'ARTIFACT_REPO', vaultKey: 'repo_path', isRequired: true],
        [envVar: 'ARTIFACT_USER', vaultKey: 'username', isRequired: true],
        [envVar: 'ARTIFACT_PASSWORD', vaultKey: 'password', isRequired: true],
    ]
]

def registrySecrets = [
    path        : 'keptn-jenkins/monaco/registry-deploy',
    secretValues: [
        [envVar: 'REGISTRY_PATH', vaultKey: 'registry_path', isRequired: true],
        [envVar: 'REGISTRY_USERNAME', vaultKey: 'username', isRequired: true],
        [envVar: 'REGISTRY_PASSWORD', vaultKey: 'password', isRequired: true]
    ]
]

def releaseToArtifactStorage(def secrets, def version, def binary) {
    withEnv(["VERSION=${version}", "BINARY=${binary}"]) {
        withVault(vaultSecrets: [secrets]) {
            sh 'curl -u "$ARTIFACT_USER":"$ARTIFACT_PASSWORD" -X PUT $ARTIFACT_REPO/monaco/$VERSION/$BINARY -T ./build/$BINARY'
        }
    }
}

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
        stage('🔍 Get current version from tag') {
            when {
                tag 'v*'
            }
            steps {
                script {
                    versionTag = sh(returnStdout: true, script: "git tag -l --points-at HEAD --sort=-creatordate | head -n 1").trim()
                    VERSION = versionTag.substring(1)  // drop v prefix
                    echo "Building release version ${VERSION}"
                }
            }
        }


        stage('🏁 Build release binaries') {
            when {
                tag 'v*'
            }
            steps {
                sh "make build-release VERSION=${VERSION}"
            }
        }

        stage('📤 Deliver release internally') {
            when {
                tag 'v*'
            }
            parallel {
                stage('🐧 Deliver Linux 32bit') {
                    steps {
                        script {
                            releaseToArtifactStorage(artifactStorageSecrets, "${VERSION}", "monaco-linux-386")
                        }
                    }
                }
                stage('🐧 Deliver Linux 64bit') {
                    steps {
                        script {
                            releaseToArtifactStorage(artifactStorageSecrets, "${VERSION}", "monaco-linux-amd64")
                        }
                    }
                }
                stage('🪟 Deliver Windows 32bit') {
                    steps {
                        script {
                            releaseToArtifactStorage(artifactStorageSecrets, "${VERSION}", "monaco-windows-386.exe")
                        }
                    }
                }
                stage('🪟 Deliver Windows 64bit') {
                    steps {
                        script {
                            releaseToArtifactStorage(artifactStorageSecrets, "${VERSION}", "monaco-windows-amd64.exe")
                        }
                    }
                }
                stage('🍏 Deliver Mac OS Apple Silicon') {
                    steps {
                        script {
                            releaseToArtifactStorage(artifactStorageSecrets, "${VERSION}", "monaco-darwin-arm64")
                        }
                    }
                }
                stage('🍏 Deliver Mac OS 64bit') {
                    steps {
                        script {
                            releaseToArtifactStorage(artifactStorageSecrets, "${VERSION}", "monaco-darwin-amd64")
                        }
                    }
                }
                stage('📦 Release Container Image') {
                    steps {
                        withEnv(["VERSION=${VERSION}"]) {
                            withVault(vaultSecrets: [registrySecrets]) {
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
