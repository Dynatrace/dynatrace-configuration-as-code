package initialise

import (
	"path/filepath"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
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

		// Now create demo files in each folder
		creator.CreateFile(nil, configTypeFolderPath, folderName, ".yaml")
		util.Log.Info("  Created File: %v/%v%v", configTypeFolderPath, folderName, ".yaml")
		creator.CreateFile(nil, configTypeFolderPath, folderName, ".json")
		util.Log.Info("  Created File: %v/%v%v", configTypeFolderPath, folderName, ".json")
	}

	return nil
}
