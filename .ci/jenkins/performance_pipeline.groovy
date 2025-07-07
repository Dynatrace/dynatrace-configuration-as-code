podTemplate(yaml: readTrusted('.ci/jenkins/agents/performance-agent.yaml')) {
    node(POD_LABEL) {
        stage("get source") {
            checkout scm
        }
        monaco = load(".ci/jenkins/tools/monaco.groovy")

        container("monaco-build") {
            stage("🏗️ building") {
                monaco.build(".")
                cleanWs()
            }
            stage("get test data") {
                git(credentialsId: 'bitbucket-buildmaster',
                    url: 'https://bitbucket.lab.dynatrace.org/scm/claus/monaco-test-data.git',
                    branch: 'main')
            }
            stage('purge tenant') {
                echo "without purge"
//                monaco.purge()
            }
        }

        try {
            container("monaco-runner") {
                stage("test") {
                    monaco.deploy("full-set", false)
                }
            }
        } finally {
            container("monaco-build") {
                stage('purge tenant') {
                    echo "without purge"
//                    monaco.purge()
                }
            }
        }
    }
}
