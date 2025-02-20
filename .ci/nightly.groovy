pipeline {
    agent {
        kubernetes {
            cloud 'linux-amd64'
            instanceCap '2'
            idleMinutes '2'
            yamlFile '.ci/jenkins_agents/ca-jenkins-agent.yaml'
            defaultContainer "ca-jenkins-agent"
        }
    }

    stages {

        stage("🌎 Integration test") {
            steps {
                executeWithSecrets(cmd: 'make integration-test')
            }
        }


        stage("📥/📤 Download/Restore test") {
            steps {
                executeWithSecrets(cmd: 'make download-restore-test')
            }
        }

        stage("🌙 Nightly Tests") {
            steps {
                executeWithSecrets(cmd: 'make nightly-test')
            }
        }

        stage("🧹 Cleanup") {
            steps {
                executeWithSecrets(cmd: 'make clean-environments')
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

def executeWithSecrets(Map args = [cmd: null]) {
    withEnv([
        "TEST_ENVIRONMENT=hardening",
        "WORKFLOW_ACTOR=bc33e56f-e8dc-4004-829b-2b02a9d77154",
    ]) {
        withVault(vaultSecrets:
            [[
                 path        : "keptn-jenkins/monaco/integration-tests/hardening",
                 secretValues: [
                     [envVar: 'OAUTH_CLIENT_ID', vaultKey: 'OAUTH_CLIENT_ID', isRequired: true],
                     [envVar: 'OAUTH_CLIENT_SECRET', vaultKey: 'OAUTH_CLIENT_SECRET', isRequired: true],
                     [envVar: 'OAUTH_TOKEN_ENDPOINT', vaultKey: 'OAUTH_TOKEN_ENDPOINT', isRequired: true],

                     [envVar: 'URL_ENVIRONMENT_1', vaultKey: 'URL_ENVIRONMENT_1', isRequired: true],
                     [envVar: 'TOKEN_ENVIRONMENT_1', vaultKey: 'TOKEN_ENVIRONMENT_1', isRequired: true],
                     [envVar: 'PLATFORM_URL_ENVIRONMENT_1', vaultKey: 'PLATFORM_URL_ENVIRONMENT_1', isRequired: true],

                     [envVar: 'URL_ENVIRONMENT_2', vaultKey: 'URL_ENVIRONMENT_2', isRequired: true],
                     [envVar: 'TOKEN_ENVIRONMENT_2', vaultKey: 'TOKEN_ENVIRONMENT_2', isRequired: true],
                     [envVar: 'PLATFORM_URL_ENVIRONMENT_2', vaultKey: 'PLATFORM_URL_ENVIRONMENT_2', isRequired: true],
                 ]
             ]]) {
            catchError(buildResult: 'FAILURE', stageResult: 'FAILURE') {
                sh(script: args.cmd)
            }
        }
    }
}
