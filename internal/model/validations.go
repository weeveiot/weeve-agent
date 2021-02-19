package model

import (
	"errors"
	"strings"
)

// ValidateManifest function to validate manifest JSON
func ValidateManifest(m Manifest) error {
	var errorList []string
	mod := m.Manifest.Search("compose").Children()
	if mod == nil {
		errorList = append(errorList, "Please provide compose")
	} else {
		net := m.Manifest.Search("compose").Search("network").Children()
		if net == nil {
			errorList = append(errorList, "Please provide network details")
		} else {
			netName := m.Manifest.Search("compose").Search("network").Search("name").Data()
			if netName == nil {
				errorList = append(errorList, "Please provide network name")
			}
		}

		mod = m.Manifest.Search("compose").Search("services").Children()

		// Check if manifest contains services
		if mod == nil || len(mod) < 1 {
			errorList = append(errorList, "Please provide at least one service")
		} else {
			for _, srv := range m.Manifest.Search("compose").Search("services").Children() {

				moduleID := srv.Search("moduleId").Data()
				if moduleID == nil {
					errorList = append(errorList, "Please provide moduleId for all services")
				}
				serviceName := srv.Search("name").Data()
				if serviceName == nil {
					errorList = append(errorList, "Please provide service name for all services")
				} else {
					imageName := srv.Search("image").Search("name").Data()
					if imageName == nil {
						errorList = append(errorList, "Please provide image name for all services")
					}

					regUrl := srv.Search("registry").Search("url").Data()
					if regUrl == nil {
						errorList = append(errorList, "Please provide registry URL for all services")
					}
				}
			}
		}
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], ","))
	} else {
		return nil
	}

}
