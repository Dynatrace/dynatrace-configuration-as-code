def credentialsEnvironment1 = [
    path        : 'keptn-jenkins/monaco/integration-tests/environment-1',
    secretValues: [
        [envVar: 'URL_ENVIRONMENT_1',   vaultKey: 'url'],
        [envVar: 'TOKEN_ENVIRONMENT_1', vaultKey: 'token']]]

def credentialsEnvironment2 = [
    path        : 'keptn-jenkins/monaco/integration-tests/environment-2',
    secretValues: [
        [envVar: 'URL_ENVIRONMENT_2',   vaultKey: 'url'],
        [envVar: 'TOKEN_ENVIRONMENT_2', vaultKey: 'token']]]

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
        stage('Build') {
            steps {
                sh "make build"
            }
        }

        stage('Tests') {
            parallel {
                stage('Unit test') {
                    steps {
                        sh "make test"
                    }
                    post {
                        always {
                            junit testResults: '**/build/test-results/test/*.xml', allowEmptyResults: true
                        }
                    }
                }

                stage('Run integration test') {
                    steps {
                        withVault(vaultSecrets: [credentialsEnvironment1, credentialsEnvironment2]) {
                            sh "make integration-test"
                        }
                    }
                }

                stage('Run integration test legacy') {
                    steps {
                        withVault(vaultSecrets: [credentialsEnvironment1, credentialsEnvironment2]) {
                            sh "make integration-test-v1"
                        }
                    }
                }

                stage('Building release binaries works') {
                    steps {
                        sh "make build-release"
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
