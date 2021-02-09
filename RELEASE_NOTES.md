# Monitoring as Code Tool - Release Notes

- Versions:
  - [1.3.1](#131)
  - [1.3.0](#130)
  - [1.2.0](#120)
  - [1.1.0](#110)
  - [1.0.1](#101)
  - [1.0.0](#100)

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
