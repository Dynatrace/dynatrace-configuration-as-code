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

        stage('Test') {
            steps {
                sh "make test"
            }
            post {
                always {
                    junit testResults: '**/build/test-results/test/*.xml', allowEmptyResults: true
                }
            }
        }

        stage('Build release binaries') {
            steps {
                sh "make build-release"
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
