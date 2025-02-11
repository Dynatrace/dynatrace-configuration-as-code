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
            stage("build log forwarder") {
                git(credentialsId: 'bitbucket-buildmaster',
                    url: 'https://bitbucket.lab.dynatrace.org/scm/claus/monaco-test-data.git',
                    branch: 'otel-log-forwarder')
                monaco.buildForwarder()
            }

            stage("get test data") {
                git(credentialsId: 'bitbucket-buildmaster',
                    url: 'https://bitbucket.lab.dynatrace.org/scm/claus/monaco-test-data.git',
                    branch: 'main')
                cleanWs()

            }
        }

        container("monaco-runner") {
            stage("test") {
                monaco.deploy("micro-set", false)
            }
        }
    }
}
