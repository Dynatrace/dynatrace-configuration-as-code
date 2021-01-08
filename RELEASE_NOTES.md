# Monitoring as Code Tool - Release Notes

- [Monitoring as Code - Release Notes](#monitoring-as-code---release-notes)
  - [1.1.0](#110)
  - [1.0.1](#101)
  - [1.0.0](#100)

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
