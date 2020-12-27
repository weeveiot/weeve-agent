package model

import (
	"errors"
	"strings"
)

func ValidateManifest(m Manifest) error {
	var errorList []string
	mod := m.Manifest.Search("compose").Children()
	if mod == nil {
		errorList = append(errorList, "Please provide compose")
	}

	mod = m.Manifest.Search("compose").Search("services").Children()

	// Check if manifest contains services
	if mod == nil || len(mod) < 1 {
		errorList = append(errorList, "Please provide at least one service")
	} else {
		for _, srv := range m.Manifest.Search("compose").Search("services").Children() {

			serviceName := srv.Search("name").Data().(string)
			if serviceName == "" {
				errorList = append(errorList, "Please provide name for all services")
			}

			imageName := srv.Search("image").Search("name").Data().(string)
			if imageName == "" {
				errorList = append(errorList, "Please provide image name for all services")
			}
		}
	}

	return errors.New(strings.Join(errorList[:], ","))
}
