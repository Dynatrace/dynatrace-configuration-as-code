def credentialsEnvironment1 = [
    path        : 'keptn-jenkins/monaco/integration-tests/environment-1',
    secretValues: [
        [envVar: 'URL_ENVIRONMENT_1',   vaultKey: 'url',    isRequired: true],
        [envVar: 'TOKEN_ENVIRONMENT_1', vaultKey: 'token',  isRequired: true]]]

def credentialsEnvironment2 = [
    path        : 'keptn-jenkins/monaco/integration-tests/environment-2',
    secretValues: [
        [envVar: 'URL_ENVIRONMENT_2',   vaultKey: 'url',    isRequired: true],
        [envVar: 'TOKEN_ENVIRONMENT_2', vaultKey: 'token',  isRequired: true]]]

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

    triggers {
        cron(env.BRANCH_NAME == 'main' ? 'H 0 * * *' : '')
    }

    stages {
        stage('🏗️ Build') {
            steps {
                sh "make build"
            }
        }

        stage('🧪 Unit test') {
            steps {
                sh "make test"
            }
            post {
                always {
                    junit testResults: '**/build/test-results/test/*.xml', allowEmptyResults: true
                }
            }
        }

        stage('🔎 Go vet') {
            steps {
                sh "make vet"
            }
        }

        stage('🚀 Binary starts') {
            steps {
                sh "make run"
            }
        }

        stage('🌎 Integration test') {
            when {
                expression {
                    env.BRANCH_IS_PRIMARY
                }
            }
            steps {
                withVault(vaultSecrets: [credentialsEnvironment1, credentialsEnvironment2]) {
                    sh "make integration-test"
                }
            }
        }

        stage('🧓 Integration test (legacy)') {
            when {
                expression {
                    env.BRANCH_IS_PRIMARY
                }
            }
            steps {
                withVault(vaultSecrets: [credentialsEnvironment1, credentialsEnvironment2]) {
                    sh "make integration-test-v1"
                }
            }
        }

        stage('🏁 Build release binaries') {
            steps {
                sh "make build-release"
            }
        }


        stage('🧹 Cleanup') {
            when {
                equals expected: true, actual: currentBuild.getBuildCauses('hudson.triggers.TimerTrigger$TimerTriggerCause').size() > 0
            }
            steps {

                withVault(vaultSecrets: [credentialsEnvironment1, credentialsEnvironment2]) {
                    sh "make clean-environments"
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
