pipeline {
    agent {
        kubernetes {
            cloud 'linux-amd64'
            instanceCap '2'
            yamlFile '.ci/jenkins_agents/ca-jenkins-agent.yaml'
            defaultContainer "ca-jenkins-agent"
        }
    }

    stages {
        stage('🚀 Release') {
            when {
                tag 'v*'
            }
            steps {
                script {
                    Context ctx

                    stage("Pre-build steps") {
                        ctx = new Context(newGithubRelease())
                        createEmptyDirectory(dir: Release.BINARIES)
                        getSignClient()
                        ctx.version = getVersionFromGitTagName(TAG_NAME)
                        if (isRelease(ctx)) {
                            ctx.githubRelease.createNew(ctx)
                            echo "GitHub release ID: ${ctx.githubRelease.releaseId}"
                        }
                    }

                    stage("Build binaries") {
                        def tasks = [:]

                        tasks["Docker container"] = { releaseDockerContainer(ctx) }

                        //linux
                        for (arch in ["amd64", "arm64", "386"]) {
                            Release r = newRelease(os: "linux", arch: arch)
                            tasks[r.binary.name()] = { releaseBinary(ctx, r) }
                        }
                        //darwin
                        for (arch in ["amd64", "arm64"]) {
                            Release r = newRelease(os: "darwin", arch: arch)
                            tasks[r.binary.name()] = { releaseBinary(ctx, r) }
                        }
                        //windows
                        for (arch in ["amd64", "386"]) {
                            Release r = newRelease(os: "windows", arch: arch)
                            tasks[r.binary.name()] = { releaseBinary(ctx, r) }
                        }

                        tasks["Generate SBOM"] = { generateSBOM(ctx) }

                        parallel tasks
                    }
                }
            }
        }
    }
}

void errorIfArgumentIsNull(Map args = [callerName: null, name: null, value: null]) {
    if (args.callerName == null) {
        error "(errorIfArgumentIsNull) method errorIfArgumentIsNull called without 'callerName' parameter"
    }
    if (args.name == null) {
        error "(errorIfArgumentIsNull) method errorIfArgumentIsNull called without 'name' parameter"
    }
    if (args.value == null) {
        error "(${args.callerName}) unspecified parameter ${args.name}"
    }
}

boolean isRelease(Context ctx) {
    return isRelease(version: ctx.version)
}

boolean isRelease(Map args = [version: null]) {
    errorIfArgumentIsNull(callerName: "isRelease", name: "version", value: args.version)
    return isReleaseCandidate(args) || isFinal(args)
}

boolean isReleaseCandidate(Map args = [version: null]) {
    errorIfArgumentIsNull(callerName: "isReleaseCandidate", name: "version", value: args.version)
    return args.version ==~ /^\d+\.\d+\.\d+-rc\.\d+$/
}

boolean isFinal(Context ctx) {
    return isFinal(version: ctx.version)
}

boolean isFinal(Map args = [version: null]) {
    errorIfArgumentIsNull(callerName: "isFinal", name: "version", value: args.version)
    return args.version ==~ /^\d+\.\d+\.\d+$/
}


/**
 * wipeDir removes specified directory if it already exists, and create new empty one
 * @param args path to a desire directory
 *      stageName: wanted name of the stage, if not provided (null) stage is not created
 */
String createEmptyDirectory(Map<String, String> args = [dir: null, stageName: null]) {
    errorIfArgumentIsNull(callerName: "createEmptyDirectory", name: "dir", value: args.dir)

    Closure action = {
        sh "rm -rf ${args.dir} && mkdir -p ${args.dir}"
    }

    if (args.stageName != null && args.stageName != "") {
        stage(args.stageName, action)
    } else {
        action.call()
    }
    return args.dir
}

/**
 * getSignClient downloads sign client
 */
void getSignClient() {
    if (!fileExists('sign-client.jar')) {
        stage("get sign-client") {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/sign-service',
                                      secretValues: [[envVar: 'repository', vaultKey: 'client_repository_url ', isRequired: true]]]]
            ) {
                sh(label: "get client", script: '''
                curl --request GET $repository/testautomation-release-local/com/dynatrace/services/signservice/client/1.0.[RELEASE]/client-1.0.[RELEASE].jar
                     --show-error
                     --silent
                     --globoff
                     --fail-with-body
                     --output sign-client.jar
               '''.replaceAll("\n", " ").replaceAll(/ +/, " "))
            }
        }
    }
}

/**
 * getVersionFromGitTagName extract version from git tag name
 */
String getVersionFromGitTagName(String tagName) {
    stage("get version") {
        // version is without the v* prefix
        def ver = tagName.substring(1)  // drop v prefix
        echo "version= ${ver}"
        return ver
    }
}


void releaseBinary(Context ctx, Release release) {
    stage("build ${release.os}/${release.arch}") {
        def knownArchs = ["arm64", "amd64", "386"]

        if (!knownArchs.contains(release.arch)) {
            error "createLinuxImage: The desired architecture ${release.arch} is not one of ${knownArchs}"
        }

        buildBinary(makeCommand: release.binary.name(), version: ctx.version, dest: release.binary.localPath())
        if (release.os == "windows") {
            signWinBinaries(source: release.binary.localPath(), version: ctx.version, destDir: '.', projectName: "monaco")
        }

        computeShaSum(source: release.binary.localPath(), dest: release.binary.shaPath())

        pushToDtRepository(source: release.binary.localPath(), dest: release.binary.dtRepositoryPath(ctx))
        pushToDtRepository(source: release.binary.shaPath(), dest: release.binary.dtRepositoryArtifactoryPathSha(ctx))

        if (isRelease(ctx)) {
            stage("push to GitHub") {
                ctx.githubRelease.addToRelease(release)
            }
        }
    }
}

void releaseDockerContainer(Context ctx) {
    createAndPublishContainer(ctx, "DT")

    if (isRelease(ctx)) {
        createAndPublishContainer(ctx, "DockerHub")

        def cosign = load(".ci/jenkins/tools/cosign.groovy")
        ctx.githubRelease.addToRelease(rawData: cosign.getPublicKey(), underName: "cosign.pub")
    }
}

void createAndPublishContainer(Context ctx, String registry) {
    def ko = load(".ci/jenkins/tools/ko.groovy")
    ko.install()
    def cosign = load(".ci/jenkins/tools/cosign.groovy")
    cosign.install("latest")

    List<String> tags = [ctx.version]
    if (isFinal(ctx)) {
        tags << "latest"
    }

    ko.loginToRegistry(registry: registry)
    image = ko.buildContainer(tags: tags, registry: registry)
    cosign.sign(image)

    echo "Created docker image ${image}"
}

void signWinBinaries(Map args = [source: null, version: null, destDir: null, projectName: null]) {
    stage('Sign binaries') {
        signWithSignService(source: args.source, version: args.version, destDir: args.destDir, projectName: args.projectName, signAction: "SIGN")
    }
}

void signWithSignService(Map args = [source: null, version: null, destDir: '.', signAction: null, projectName: null]) {
    withEnv(["source=${args.source}", "version=${args.version}", "destDir=${args.destDir}", "signAction=${args.signAction}", "project=${args.projectName}"]) {
        withVault(vaultSecrets: [[path        : 'signing-service-authentication/monaco',
                                  secretValues: [
                                      [envVar: 'username', vaultKey: 'username', isRequired: true],
                                      [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
        ) {
            sh '''
                java -jar sign-client.jar
                  --action $signAction
                  --digestAlg SHA256
                  --downloadPath $destDir
                  --kernelMode false
                  --triggeringProject $project
                  --triggeringVersion $version
                  --username $username
                  --password $password
                  $source
               '''.replaceAll("\n", " ").replaceAll(/ +/, " ")
        }
    }
}

void buildBinary(Map args = [makeCommand: null, version: null, dest: null]) {
    stage("build binaries") {
        sh "make ${args.makeCommand} VERSION=${args.version} OUTPUT=${args.dest}"
    }
}

void computeShaSum(Map args = [source: null, dest: null]) {
    stage("compute sha256 sum") {
        sh "sha256sum ${args.source} | sed 's, .*/,  ,' > ${args.dest}"
        sh "cat ${args.dest}"
    }
}

def pushToDtRepository(Map args = [source: null, dest: null]) {
    stage('push to repository') {
        withEnv(["source=${args.source}", "dest=$args.dest"]) {
            withVault(vaultSecrets: [[path        : 'keptn-jenkins/monaco/artifact-storage-deploy',
                                      secretValues: [
                                          [envVar: 'repo_path', vaultKey: 'repo_path', isRequired: true],
                                          [envVar: 'username', vaultKey: 'username', isRequired: true],
                                          [envVar: 'password', vaultKey: 'password', isRequired: true]]]]
            ) {
                sh(label: "push to DT artifactory", script: '''
                    curl --request PUT $repo_path/$dest
                         --user "$username":"$password"
                         --fail-with-body
                         --upload-file $source
                   '''.replaceAll("\n", " ").replaceAll(/ +/, " "))
            }
        }
    }
}

void generateSBOM(Context ctx) {
    stage("Generate SBOM") {
        final String sbomName = "sbom.cdx.json"

        def cyclonedxGomod = load(".ci/jenkins/tools/cyclonedxGomod.groovy")
        cyclonedxGomod.install("latest")

        cyclonedxGomod.generateSbom(sbomName)
        pushToDtRepository(source: sbomName, dest: "monaco/${ctx.version}/${sbomName}")
        if (isRelease(ctx)) {
            ctx.githubRelease.addFileToRelease(filename: sbomName, underName: sbomName)
        }
    }
}


class Context {
    String version
    GithubRelease githubRelease

    Context(def githubRelease) {
        this.githubRelease = githubRelease
    }
}


GithubRelease newGithubRelease() {
    Map args = [:]
    args.githubTools = load(".ci/jenkins/tools/github.groovy")
    return new GithubRelease(args)
}

class GithubRelease {
    private githubTools
    String releaseId

    GithubRelease(Map args = [githubTools: null]) {
        githubTools = args.githubTools
    }

    String createNew(Context ctx) {
        this.releaseId = githubTools.createRelease(version: ctx.version)
        return this.releaseId
    }

    void addToRelease(Release release) {
        githubTools.pushFileToRelease(rleaseName: release.binary.name(), filename: release.binary.localPath(), releaseId: this.releaseId)
        githubTools.pushFileToRelease(rleaseName: "${release.binary.shaName()}", filename: release.binary.shaPath(), releaseId: this.releaseId)
    }

    void addFileToRelease(Map args = [filename: null, underName: null]) {
        githubTools.pushFileToRelease(rleaseName: args.underName, filename: args.filename, releaseId: this.releaseId)
    }

    void addToRelease(Map args = [rawData: null, underName: null]) {
        githubTools.pushRawDataToRelease(rleaseName: args.underName, rawData: args.rawData, releaseId: this.releaseId)
    }
}

Release newRelease(Map args = [os: null, arch: null]) {
    errorIfArgumentIsNull(callerName: "newRelease", name: "os", value: args.os)
    errorIfArgumentIsNull(callerName: "newRelease", name: "arch", value: args.arch)

    def knownOs = ["linux", "darwin", "windows"]
    def knownArchs = ["arm64", "amd64", "386"]

    if (!knownOs.contains(args.os)) {
        error "newRelease: The desired OS ${args.os} is not one of ${knownOs}"
    }
    if (!knownArchs.contains(args.arch)) {
        error "newRelease: The desired architecture ${args.arch} is not one of ${knownArchs}"
    }

    new Release(args)
}

class Release {
    static final String BINARIES = './bin'
    private String os //e.g. linux, windows, darwin
    private String arch //e.g. 386, amd64, arm64
    Binary binary

    Release(Map args = [os: null, arch: null]) {
        this.arch = args.arch
        this.os = args.os

        this.binary = new Binary()
    }

    class Binary {
        String name() {
            if (os == "windows") {
                return "monaco-${os}-${arch}.exe"
            }
            return "monaco-${os}-${arch}"
        }

        String shaName() {
            return "${binary.name()}.sha256"
        }

        String localPath() {
            return "${BINARIES}/${binary.name()}"
        }

        String shaPath() {
            return "${BINARIES}/${binary.shaName()}"
        }

        String dtRepositoryPath(Context ctx) {
            return "monaco/${ctx.version}/${binary.name()}"
        }

        String dtRepositoryArtifactoryPathSha(Context ctx) {
            return "monaco/${ctx.version}/${binary.shaName()}"
        }
    }
}




