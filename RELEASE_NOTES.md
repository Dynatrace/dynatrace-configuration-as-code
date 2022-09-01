# Monitoring as Code Tool - Release Notes

- Versions:
  - [1.8.3](#183)
  - [1.8.2](#182)
  - [1.8.1](#181)
  - [1.8.0](#180)
  - [1.7.0](#170)
  - [1.6.0](#160)
  - [1.5.3](#153)
  - [1.5.2](#152)
  - [1.5.1](#151)
  - [1.5.0](#150)
  - [1.4.0](#140)
  - [1.3.1](#131)
  - [1.3.0](#130)
  - [1.2.0](#120)
  - [1.1.0](#110)
  - [1.0.1](#101)
  - [1.0.0](#100)

## 1.8.3
### List of changes
#### Bug fixes

* 6e0016 fix: Do not abort download after the first download error.

## 1.8.2
### List of changes
#### Features
* 086ebb5 Retry SLO creation on additional know error

#### Bug fixes
* 1d94584 Workaround synthetic API failing if Application ID can not (yet) be found
* e090000 filter the mime types correctly for logging (#709)
* 7b3ad72 (download) Deleting configs with slashes in the name does no longer error. (#706)
* c707483 (download) Downloading single config Apis now report encountered errors

#### Documentation
* 04807b4 Update download link to latest monaco release
* 373b8d8 Rename 1.8.0 sidebar to 1.8.1 as .0 docs where completely replaced
* ad702cc Upgrade to docusaurus 2.0.1 release

## 1.8.1

### List of changes

#### Misc changes
* d920125 ci: Build static monaco binary
* 414cc2d ci: Run build-release for each build
* 7b13c4e ci: Remove no longer needed xgo github action step

## 1.8.0

### List of changes

#### Features
* 62e3862 feat(yaml_creator): order YAML-files Lexographically
* 1acb1e7 feat: Allow newline characters in variables
* 8808ab6 feat: Improve error handling and logging of Delete operation
* 6e78134 feat: Improve some log messages (#685)
* 2459391 feat: Add support for elasticsearch-extension (#677)
* de9304b feat: Handle Dynatrace Configurations with non-unique names
* 191458b feat: Also recognize .yml files as YAML
* 78d5ff8 feat: Download non Dynatrace preset dashboards (#644)
* dc9d4de feat: Support 'Single Configuration' APIs
* 010c23e Feature/610/reports endpoint (#614)
* dd16c28 feat: remove metadata
* 4006c75 feat: add endpoints for disk anomaly detection
* 55bf694 feat(workflow): Add CODEOWNERS default reviewers
* 241a4c0 feat(parse): produce more detailed error messages for YAML conversion issues


#### Bugfixes
* 2075820 fix(download): added filter for extensions
* ff6e33a fix: Log JsonValidationErrors that do not have line information
* ca745fc fix(download): Remove non standard identifier `entityId` during download
* ddbc836 fix(download): Remove standard identifiers from configs during download
* 109fce8 fix: added unit test
* a840d83 fix: download-creates-folders-for-empty-configurations
* 3ad8aad fix: incorptorated pullrequest #445* 
* 2029bc8 fix: download help descriptions fixed
* 4ea4135 fix: changed flag `downloadSpecificAPI` to `specific-api`
* 33547b3 fix: Check for "Unknown management zone" (#651)
* 7997ac4 fix: do not ignore vars that have "=" in the value

#### Misc changes
* 0d900d4 refactor: Remove unused symbols (#681)
* 05eb9b1 test: skip dependency transformation (#612)
* b73404d test: fix filter type
* c80eb41 build: update all runners to go ~1.16
* db45ac8 build: restrict go version to >= 1.16 and < 1.17
* 7256eca test(parse): add tests for failing YAML conversion


#### Documentation 
* 36c1e22 docs: Minor fixes & rewordings to the documentation (#682)
* bd8877b docs: Correct filepath in validation-configuration.md sample
* ef36def docs: Fix case
* 31618f5 docs: Fix case
* afa2e39 docs: Resolve mismatch between latest and 1.7.0
* 70e6b4b docs: update macos installation url
* 19ced19 docs: fix macos download url
* 66d4cb8 docs: remove duplicate text, reword what's left. (#511)
* f98975c docs: Minor wordsmithing (#512)
* de9562e docs: Minor wordsmithing to "What is Monaco" (#513)
* 364fb5d docs: Minor edit to "Add API" (#514)
* 64404bf docs: trivial edits for your consideration (#541)
* ceb86c8 docs: trivial edits for your consideration (#542)
* 3fa52b9 docs: small edits for your consideration (#543)
* 8cc35d9 docs: editorial changes to "New CLI" (#544)
* a917845 docs: trivial edits to "Download configuration" (#545)
* d57aa10 docs: edits to Logging.md (#546)
* e610800 docs: minor edits to "Configuration structure" (#547)
* 09d8ef2 docs: edits to yaml_config.md (#548)
* 86462e5 docs: edits to plugin_config.md (#549)
* 2a4a1b8 docs: edits to delete_config.md (#550)
* 76f9fc6 docs: edits to add_new_api.md (#551)
* 8ade72d docs: edits to bill of materials.md (#552)
* 24fcbb2 Documentation fixes for 1.7 (#540)


#### Library updates
* ed51ede chore(deps): bump github.com/urfave/cli/v2 from 2.11.0 to 2.11.1
* e56d5b7 chore(deps): bump terser from 4.8.0 to 4.8.1 in /documentation
* 46955f6 chore(deps): bump github.com/spf13/afero from 1.9.0 to 1.9.2 (#688)
* e40e584 chore(deps): bump github.com/spf13/afero from 1.8.2 to 1.9.0 (#680)
* 443426e chore(deps): bump crazy-max/ghaction-xgo from 2.0.0 to 2.1.0 (#684)
* a9167c8 chore(deps): bump github.com/urfave/cli/v2 from 2.8.1 to 2.11.0
* d5f0bed chore(deps): bump eventsource from 1.1.0 to 1.1.2 in /documentation
* 3164b7e chore(deps): bump crazy-max/ghaction-xgo from 1.8.0 to 2.0.0
* cac89d7 chore(deps): bump github.com/urfave/cli/v2 from 2.7.1 to 2.8.1
* 2cdc845 ci(dependabot): Drop dependency updates for next branch
* 4e4b71d chore(deps): bump github.com/urfave/cli/v2 from 2.6.0 to 2.7.1
* 5bd0f8f chore(deps): bump crazy-max/ghaction-xgo from 1.6.1 to 1.8.0
* ab03369 chore(deps): bump actions/checkout from 2.4.0 to 3.0.2
* abe8dbf chore(deps): bump github.com/urfave/cli/v2 from 2.3.0 to 2.6.0
* 140d6e5 chore(deps): bump github.com/google/go-cmp from 0.5.7 to 0.5.8
* 815b28a chore(deps): bump actions/setup-go from 2 to 3
* 226151c chore(deps): bump actions/upload-artifact from 2 to 3
* b57c5a9 chore(deps): bump minimist from 1.2.5 to 1.2.6 in /documentation
* bf5ed4d chore(deps): bump async from 2.6.3 to 2.6.4 in /documentation
* 01ba113 chore(deps): bump github.com/spf13/afero from 1.6.0 to 1.8.2
* b1f58e9 chore(deps): bump golangci/golangci-lint-action from 3.1.0 to 3.2.0
* 0b2776b chore(deps): bump golangci/golangci-lint-action from 2.5.2 to 3.1.0
* f29b07d chore(deps): bump actions/setup-node from 2.5.1 to 3
* 6ba8b7e chore(deps): bump prismjs from 1.26.0 to 1.27.0 in /documentation
* 22b1fd7 chore(deps): bump github.com/google/go-cmp from 0.5.6 to 0.5.7
* 898e2a6 chore(deps): bump url-parse from 1.5.4 to 1.5.10 in /documentation
* b8fadb0 chore(deps): bump actions/setup-node from 2.5.0 to 2.5.1


## 1.7.0

### List of changes

#### Features
* f42141a feat: print warning if two or mode configurations having same name are detected in Dynatrace (#357)
* 51d83f3 feat: Check Version before Extension upload (#386)
* debc4cb feat(workflow): Add workflow for generating bill-of-materials (#471)
* c8a9f7c feat(deps): Add the next branch to dependabot targets (#470)
* 4e5c6be feat(api): Added service detection APIs (#499)
* 0554279 feat(api): calculated metrics for mobile, web, synthetic, service (#502)
* 1611e13 feat(util): Introduce UUID v3 generator util (#538)
* 27381b9 feat(rest): Add util to check Dynatrace version (#538)

#### Bugfixes
* 525f69e fix: update vulnerable security packages (#381)
* 21feb04 fix: Check status code in extension validation to allow uploading new extensions again (#392)
* d3287f7 fix: upgrade packages with security alerts (#420)
* f1dbd48 fix(integration tests): Fix breaking chnge in SLO api (#485)
* bd5b8b8 fix(api): Always include query param for anomaly detection metrics API (#468)
* c4855d7 fix: do not ignore non-successful responses on GET(all) requests (#503)
* 5f58408 fix(rest): Handling timing issues on Dynatrace object creation (#538)
* d7cddde fix(rest): Strip create only property from application-mobile on update (#538)

#### Misc changes
* a2f58cd fix: add prefix to dependabot commit messages to comply with (#361)
* (several) refactor(rest): Structure REST code (#538)
* 4e1bdaa refactor(rest): Build full URL in upsert method rather than caller (#538)
* 02b088d ci: Activate verbose logging in intergation tests (#538)
* 45e8a06 ci: Update Mobile Application ID in integration test config (#538)

#### Documentation 
* 1156003 docs: Adjust version in linux installation instructions (#362)
* 5a7c00d docs: Removed duplicate lines in yaml_config.md
* a3f8a35 docs: Adding a note about using --dry-run with the new CLI. (#355)
* 98311d6 docs: Add README on how to contribute to the documentation (#388)
* 4966b26 docs: update README with bill of materials (#481)
* 5791b49 docs: add Bill of materials and license (#489)
* 777e9b3 docs: fix typos in downloading-configuration.md (#458)
* 26a6322 docs: update documentation for 1.7 release (#539)

#### Library updates
* da5c405 chore(deps): bump actions/setup-node from 2.1.5 to 2.2.0
* ebdd4d1 chore(deps): Bump prismjs from 1.23.0 to 1.24.0 in /documentation
* 6b3ecc3 chore(deps): bump github.com/google/uuid from 1.2.0 to 1.3.0
* 05d183b chore(deps): bump actions/setup-node from 2.2.0 to 2.4.0
* 1d43129 chore(deps): bump actions/checkout from 2.3.4 to 2.3.5
* 3186ff5 chore(deps): bump actions/setup-node from 2.4.0 to 2.4.1 (#434)
* 8a02a9d chore(deps): bump actions/checkout from 2.3.5 to 2.4.0
* 5a73b53 chore(deps): bump actions/setup-node from 2.4.1 to 2.5.0
* 70c3338 chore(deps): bump algoliasearch-helper in /documentation

## 1.6.0

### List of changes

#### Features
* 04ec13c feat(api): add failuredetection api and fix broken contribution link (#283)
* 67b163d feat: Remove unsupported macOS 32bit build target
* ef01727 feat: Build monaco releases for macOS M1/arm

#### Bugfixes
* 1baf806 fix: Upsert objects with special chars no longer fail

#### Documentation 
* (several): Add Github Pages Documentation 
* 8fa751a doc: Fix install section in README

#### Library updates
* 15b650b Bump github.com/golang/mock from 1.5.0 to 1.6.0
* 93eefbf Bump actions/setup-node from 1 to 2.1.5
* d0440dd Bump github.com/google/go-cmp from 0.5.5 to 0.5.6
* a7b93c4 Bump actions/checkout from 2 to 2.3.4
* 8c610cf Bump actions/create-release from 1.0.0 to 1.1.4
* ac72312 Bump crazy-max/ghaction-xgo from 1 to 1.6.1
* f6b9fd2 Bump golangci/golangci-lint-action from v2 to v2.5.2

## 1.5.3

### List of changes

#### Bugfixes
* 7b0cfae Bugfix #307 multiple dependencies with relative path resolution not working


## 1.5.2

### List of changes

#### Bugfixes
* 422679b fix: Update mobile application config to fix integration tests (#289)
* b3761ee fix: References on pre-existing SLOs do not work (#289)

#### Misc changes
* f5303a7 ci: Add static code analysis of PRs (#286)

## 1.5.1

### List of changes

#### Bugfixes
* bb125fb fix: Paging in monaco was not working (#276)
* 7388bbd Fix absolute paths broken
* 7bd5a7f Fix template loading for absolute template reference
* affafab reset env variable at the end of e2e test (#266)
* 559b881 removes fs from config implementation (#266)
* 397a8b2 fix parameters in integration test (#266)
* 2440ee7 e2e test config restore environment (#266)

#### Misc changes
* 7e38119 Add licensecheck dependencies to go mod (#250)
* 72721cf Handle possible error when creating a new go http request
* cea13c5 Do not unmarshall json before sending it to api
* 0031be2 Upgrade to go 1.16
* 73c3751 afero as filesystem
* ace1513 Improve check-license-headers.sh script

#### Library updates
* dd0ce7d Bump github.com/google/go-cmp from 0.5.4 to 0.5.5
* aaf8c64 Bump github.com/spf13/afero from 1.5.1 to 1.6.0

#### Documentation
* a789838 Added documentation about custom extensions #113 (#251)
* fc09d97 Added proxy documentation (#249)
* e486bf5 Added dashboards documentation (#247)

## 1.5.0

### List of changes
#### Features
* 84c8ef9 Added option to allow deployment to proceed even if error occurred (#127)
* c695dd4 Add response log (#228)

#### Bugfixes
* 7159440 Add missing license headers to files (#191)
* 2462ba0 Fix dependency resolution not working if defined in overrides (#141)

#### Library updates
* e217ea8 Bump github.com/golang/mock from 1.4.4 to 1.5.0

#### Misc changes
* e178bc5 Fix path of PUT in how-to-add-a-new-api.md (#235)
* 0faae65 Add missing go mod entries for addlincense
* 080f2f1 Update README.md Updated supported configuration types table
* faa1b01 Add add-license-header make goal
* ac6f001 Fail build if license header is missing

## 1.4.0

### List of changes
#### Features

* f5a4f71 Add support for credentials vault API (#175)
* af5c4f5 Fix #158: Add support for custom-service APIs for dotnet, go, nodejs, and php (#172)

## 1.3.1

### List of changes
#### Bugfixes

* 2e0e88a: Cope with faulty configs (name or id is null) (#169)

## 1.3.0

### List of changes

#### Deprecation Warnings
* Fix #146: `application-web` configuration type replaces `application` in the future (#149)
* `application` config type is deprecated with 1.3.0 and will be removed with 2.0.0 (#149)

#### Features

* Fix #146: Add support for `application-mobile`(applications/mobile) configurations (#149)
* 5558b61 Fix #66: Add support for `slo` (/api/v2/slo) configruations (#153)

#### Bugfixes

* f52d297 Fix #138: Fix issue when providing context path without slash (#139)
* 60cf6b1 Fix #155: Fix delete command not working (#157)
* 4ad81a9 Fix #102: Trim URL Trailing Slashes from environment URLs (#142)
* 2f774bc: Show error when more than one argument is provided (#139)
* 7002149: Fix pretty print of json error (#150)
* 4086d87: Fix verbose flag not working in new cli (#159)
* d911e21: Fix typos in CLI usage info messages (#159)

#### Misc changes

* 27a154c: Add possibility to log requests sent to Dynatrace (#151)
  * To aid debugging it is now possible to have monaco log what exactly it sends to Dynatrace (fully filled out config JSONs)
* 3772697: Replace json unmarshall type with map - improves internal handling of responses from Dynatrace (#150)

## 1.2.0

### List of changes

#### Features

* 4c53146 and 8611bd5 #19: Download current environment configuration (#122) 
* dfbad92 and f54c278 Fix #45: Introduce new experimental cli design (#100)

#### Bugfixes

* fe30566 Fix #136: Handle failing to find project-root-folder path (#137)
* 211dcbe and b74b06b Fix missing verbose flag in legacy cli (#129)

#### Misc changes

* c70ca7b Fix #82: Define some install steps/requirements (#109)

## 1.1.0

### List of changes

#### Features

* ee01baf Fix #77: Enhance log messages for syntax errors detected in JSONs (#88)
* 2dc498d Fix #87: Implement simple strategy to prevent running into rate limit (#89)
* 42cb186 Encapsulate REST code in DynatraceClient (#74)
* cfe9179 Fix #84: Add --version flag (#95)
* 5f82131 Fix #40 and #75: Improve help text (#75)
* 75213b9 Fix #58: Log a warning if project name clashes with API name (#58)
* af1bfba Fix #68: Update Request Naming Folder Name to request-naming-service (#73)
* b0d4f44 Fix #68: Add Global Service Request Naming Support (#69)

#### Bugfixes

* e09bde2 Fix #60: Allow path separators in variables (#83)
* c187ffc Fix #80 and #93: Allow trailing slash in folder on Windows (#94)
* 82bb035 Fix #37: Add Windows handling to Make targets (#72)
* c0f31df Fix #20: Delete configs only if apply finished without errors (#79)
* 850c0de Fix #52: Fix unresolved references errors in GoLand (#53)
* a6b6c1d Fix #41: Fixes environment-url key in README.md (#49)
* 4bb6100 Fix #22: Fix issue that loads properties from configs that start with the same name (#35)

#### Library updates

* 1f0fa25 Bump gopkg.in/yaml.v2 from 2.2.5 to 2.4.0 
* dff40d7 Bump github.com/google/go-cmp from 0.5.3 to 0.5.4 

#### Misc changes

* b86ffb3 Fix #85: Update README delete section (#99)
* 946d8ed Execute build as default make command (#96)
* 793f029 Fix whitespace in README.md (#81)
* a14f1a3 Fix #54: Enhance documentation with Token permissions (#70)
* a8c6522 Add feature request template (#50)
* 1b962f2 Fix #63: Fix the synthetic location id used in the integration tests (#64)
* 5e29711 Extend Documentation of Calculated Log Metrics (#56)

## 1.0.1
* 75204ce Fix #43: Fix Logger Setup. Fixes verbose logging flag being ignored and file logs not being written. (#44)

## 1.0.0
*Initial release*
* Create, update and delete monitoring configuration on Dynatrace environments
