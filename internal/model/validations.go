package model

import (
	"errors"
	"strings"

	"github.com/Jeffail/gabs/v2"
)

// ValidateManifest function to validate manifest JSON
func ValidateManifest(m Manifest) error {
	var errorList []string

	id := m.Manifest.Search("id").Data()
	if id == nil {
		errorList = append(errorList, "Please provide data service id")
	}
	version := m.Manifest.Search("version").Data()
	if version == nil {
		errorList = append(errorList, "Please provide data service version")
	}
	name := m.Manifest.Search("name").Data()
	if name == nil {
		errorList = append(errorList, "Please provide data service name")
	}
	services := m.Manifest.Search("services").Children()
	// Check if manifest contains services
	if services == nil || len(services) < 1 {
		errorList = append(errorList, "Please provide at least one service")
	} else {
		for _, srv := range services {
			moduleID := srv.Search("id").Data()
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
			}
		}
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], ","))
	} else {
		return nil
	}

}

func ValidateStartStopJSON(jsonParsed *gabs.Container) error {
	var errorList []string
	serviceID := jsonParsed.Search("id").Data()
	if serviceID == nil {
		errorList = append(errorList, "Expected Data Service ID 'id' in JSON, but not found.")
	}
	serviceName := jsonParsed.Search("name").Data()
	if serviceName == nil {
		errorList = append(errorList, "Expected Data Service Name 'name' in JSON, but not found.")
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList[:], " "))
	} else {
		return nil
	}
}
