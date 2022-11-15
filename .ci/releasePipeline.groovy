def artifactoryCredentials = [
    path        : 'keptn-jenkins/monaco/artifactory-deploy',
    secretValues: [
        [envVar: 'ARTIFACTORY_USER', vaultKey: 'username', isRequired: true],
        [envVar: 'ARTIFACTORY_PASSWORD', vaultKey: 'password', isRequired: true],
    ]
]

def harbor = [
    registry   : 'registry.lab.dynatrace.org/monaco',
    credentials: [
        path        : 'keptn-jenkins/monaco/harbor-account',
        secretValues: [
            [envVar: 'username', vaultKey: 'username', isRequired: true],
            [envVar: 'password', vaultKey: 'password', isRequired: true]
        ]
    ]
]

def releaseToArtifactory(def credentials, def version, def binary) {
    withEnv(["VERSION=${version}", "BINARY=${binary}"]) {
        withVault(vaultSecrets: [credentials]) {
            sh 'curl -u "$ARTIFACTORY_USER":"$ARTIFACTORY_PASSWORD" -X PUT https://artifactory.lab.dynatrace.org/artifactory/monaco-local/monaco/$VERSION/$BINARY -T ./build/$BINARY'
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

        stage('📤 Deliver release to Artifactory') {
            when {
                tag 'v*'
            }
            parallel {
                stage('🐧 Deliver Linux 32bit') {
                    steps {
                        script {
                            releaseToArtifactory(artifactoryCredentials, "${VERSION}", "monaco-linux-386")
                        }
                    }
                }
                stage('🐧 Deliver Linux 64bit') {
                    steps {
                        script {
                            releaseToArtifactory(artifactoryCredentials, "${VERSION}", "monaco-linux-amd64")
                        }
                    }
                }
                stage('🪟 Deliver Windows 32bit') {
                    steps {
                        script {
                            releaseToArtifactory(artifactoryCredentials, "${VERSION}", "monaco-windows-386.exe")
                        }
                    }
                }
                stage('🪟 Deliver Windows 64bit') {
                    steps {
                        script {
                            releaseToArtifactory(artifactoryCredentials, "${VERSION}", "monaco-windows-amd64.exe")
                        }
                    }
                }
                stage('🍏 Deliver Mac OS Apple Silicon') {
                    steps {
                        script {
                            releaseToArtifactory(artifactoryCredentials, "${VERSION}", "monaco-darwin-arm64")
                        }
                    }
                }
                stage('🍏 Deliver Mac OS 64bit') {
                    steps {
                        script {
                            releaseToArtifactory(artifactoryCredentials, "${VERSION}", "monaco-darwin-amd64")
                        }
                    }
                }
            }
        }
        stage('📦 Release Container Image') {
            when {
                tag 'v*'
            }
            steps {
                withEnv(["VERSION=${VERSION}", "REGISTRY=${harbor.registry}"]) {
                    withVault(vaultSecrets: [harbor.credentials]) {
                        script {
                            sh 'docker login $REGISTRY -u "$username" -p "$password" '
                            sh 'docker build -t $REGISTRY/monaco/dynatrace-monitoring-as-code:$VERSION .'
                            sh 'docker push $REGISTRY/monaco/dynatrace-monitoring-as-code:$VERSION'
                        }
                    }
                }
            }
            post {
                always {
                    withEnv(["REGISTRY=${harbor.registry}"]) {
                        sh 'docker logout $REGISTRY'
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
