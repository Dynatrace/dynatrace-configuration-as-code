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

void buildMonaco() {
    stage('build monaco') {
        sh "CGO_ENABLED=0 go build -buildvcs=false -a -tags netgo -ldflags \"-X github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version.MonitoringAsCode=2.x -w -extldflags -static\" -o $JENKINS_AGENT_WORKDIR/monaco ./cmd/monaco"
    }
}


void monacoBuild(String sourcePath) {
    String monacoBin = "${JENKINS_AGENT_WORKDIR}/monaco"
    sh(label: "build monaco",
        script: """CGO_ENABLED=0
                go build
                  -a -tags=netgo -buildvcs=false
                  -ldflags=\"-X github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version.MonitoringAsCode=2.x -w -extldflags -static\"
                  -o ${monacoBin}
                  ${sourcePath}/cmd/monaco
            """.replaceAll("\n", " ").replaceAll(/ +/, " "))

}

void monacoPurge() {
    String monacoBin = "${JENKINS_AGENT_WORKDIR}/monaco"
    sh(label: "purge tenant",
        script: "MONACO_ENABLE_DANGEROUS_COMMANDS=true ${monacoBin} purge --help")
}

void monacoDeploy() {
    String monacoBin = "${JENKINS_AGENT_WORKDIR}/monaco"
    sh(label: "monaco deploy",
        script: " ${monacoBin} deploy --help")
}
