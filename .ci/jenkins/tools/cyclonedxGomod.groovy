/*
 * @license
 * Copyright 2024 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


/**
 * install sets the required tools
 */
void install(String version) {
    sh(label: "download cyclonedx-gomod", script: "go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@${version}")
}

void generateSbom(String fileName) {
    sh(label: "generate SBOM", script: "cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/monaco/ -output ${fileName}")
}

return this
