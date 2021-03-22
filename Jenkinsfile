pipeline {
    agent any
    stages {
        stage('Stage 1') {
            steps {
                checkout([$class: 'GitSCM', branches: [[name: 'main']], extensions: [], userRemoteConfigs: [[url: 'https://github.com/rszulgo/dynatrace-monitoring-as-code.git']]])
            }
        }
    }
}
