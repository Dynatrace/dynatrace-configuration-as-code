package initialise

import (
	"path/filepath"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/yamlcreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
)

//CreateTemplate Creates a blank set of monaco folders and files for each supported API
func CreateTemplate(workingDir string) error {

	workingDir = filepath.Clean(workingDir)
	util.Log.Info("Initialising Monaco Demo Folders")
	projectsFolder := workingDir + "/projects"
	demoProjectFolder := projectsFolder + "/baseconfig"

	creator := files.NewDiskFileCreator()

	// Create environments.yaml file
	environmentsContent := `environment1:
    - name: "environment1"
    - env-url: "{{ .Env.ENVIRONMENT_ONE }}"
    - env-token-name: "{{ .Env.TOKEN_ENVIRONMENT_ONE }}"`
	_, err := creator.CreateFile([]byte(environmentsContent), workingDir, "environments", ".yaml")

	if err != nil {
		util.Log.Error("Error creating environments.yaml file")
	}
	util.Log.Info("Created File: environments.yaml")

	// Create /projects folder
	fullpath, err := creator.CreateFolder(projectsFolder)
	if err != nil {
		util.Log.Error("Error creating top level projects folder. %v - %v", fullpath, err)
		return err
	}
	util.Log.Info("Created Folder: %v", projectsFolder)

	// Create /projects/baseconfig folder
	creator.CreateFolder(demoProjectFolder)
	if err != nil {
		util.Log.Error("Error creating top level projects folder. %v - %v", fullpath, err)
		return err
	}
	util.Log.Info("Created Folder: %v", demoProjectFolder)

	// For each allowed API, create the relevant folder
	// Retrieve all allowed APIs
	apiMap := api.NewApis()

	for folderName := range apiMap {
		configTypeFolderPath := demoProjectFolder + "/" + folderName
		creator.CreateFolder(configTypeFolderPath)
		if err != nil {
			util.Log.Error("Error creating /projects/%v >> %v - %v", folderName, fullpath, err)
			return err
		}
		util.Log.Info("Created Folder: %v", configTypeFolderPath)

		config := yamlcreator.NewYamlConfig()
		/* Add config like:
				*  config:
				*  - folderName: folderName.json
				*  folderName:
				*  - name: folderName-one
				*
				* For example:
		        * config:
		        * - alerting-profile: alerting-profile.json
		        * alerting-profile:
		        * - name: alerting-profile-one
		*/
		config.AddConfig(folderName, folderName+"-one")
		fileCreator := files.NewDiskFileCreator()
		/* Create file as "folderName.yaml"
		 * eg. alerting-profile.yaml
		 */
		err := config.CreateYamlFile(fileCreator, configTypeFolderPath, folderName)
		if err != nil {
			util.Log.Error("Error creating YAML file %v", err)
		}

		util.Log.Info("  Created File: %v/%v%v", configTypeFolderPath, folderName, ".yaml")
		creator.CreateFile(nil, configTypeFolderPath, folderName, ".json")
		util.Log.Info("  Created File: %v/%v%v", configTypeFolderPath, folderName, ".json")
	}

	return nil
}
