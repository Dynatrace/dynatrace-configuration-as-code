pipeline {
    //Setting the environment variables DISABLE_AUTH and DB_ENGINE
    environment {
        TOKEN_ENVIRONMENT_1=credentials('innov-token')
        NEW_CLI= 1
    }

    agent any
    stages {
        stage('Checkout') {
            steps {
                checkout([$class: 'GitSCM', branches: [[name: 'main']], extensions: [], userRemoteConfigs: [[url: 'https://github.com/rszulgo/dynatrace-monitoring-as-code.git']]])
            }
        }
        stage('Check style') {
            steps {
                sh 'make lint'
            }
        }
        stage('Build') {
            steps {
                sh 'make build'
            }
        }
        stage('Test') {
            steps {
                sh 'make test'
            }
        }
        stage('Run Monaco') {
            steps {
                sh './bin/monaco deploy --verbose --project innov2021 --environments projects/environments.yaml --specific-environment innov2021 projects/'
            }
        }
    }
}
