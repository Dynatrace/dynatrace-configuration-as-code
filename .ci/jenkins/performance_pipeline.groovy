podTemplate(cloud: 'linux-amd64-injected', yaml: readTrusted('.ci/jenkins/agents/performance-agent.yaml')) {
    node(POD_LABEL) {
        stage("get source") {
            checkout scm
        }
        monaco = load(".ci/jenkins/tools/monaco.groovy")
        commitHash = monaco.getCommitHash()

        container("monaco-build") {
            stage("🏗️ building") {
                monaco.build(".")
                cleanWs()
            }
            /*
            stage("build log forwarder") {
                git(credentialsId: 'bitbucket-buildmaster',
                    url: 'https://bitbucket.lab.dynatrace.org/scm/claus/monaco-test-data.git',
                    branch: 'generic-log-ingester')
                monaco.buildForwarder()
            }*/

            stage("get test data") {
                git(credentialsId: 'bitbucket-buildmaster',
                    url: 'https://bitbucket.lab.dynatrace.org/scm/claus/monaco-test-data.git',
                    branch: 'main')
                monaco.copyTestData()
                cleanWs()
            }
        }

        container("monaco-runner") {
            // stage("install LibC6Compat") {
            //     monaco.installLibC6Compat()
            // }
            stage("test") {
                monaco.deploy("small-set", commitHash, false)
            }
        }
    }
}
