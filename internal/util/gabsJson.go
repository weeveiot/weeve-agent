package util

import (
	"fmt"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

func ParseImagesList(manifestJson []byte) []string {
	jsonParsed, err := gabs.ParseJSON(manifestJson)
	if err != nil {
		panic(err)
	}
	var imageNamesList []string
	for _, mod := range jsonParsed.Search("Modules").Children() {
		imageNamesList = append(imageNamesList, mod.Search("ImageName").Data().(string))
	}
	return imageNamesList
}

// Print a manifest object as defined
func PrintManifestDetails(json []byte) bool {
	jsonParsed, err := gabs.ParseJSON(json)
	if err != nil {
		panic(err)
	}

	log.Debug("Manifest id: ", jsonParsed.Search("ID").Data())
	log.Debug("Manifest name: ", jsonParsed.Search("Name").Data())
	log.Debug("Modules:")
	for _, mod := range jsonParsed.Search("Modules").Children() {
		log.Debug(fmt.Sprintf("\tindex: %v, name: %v", mod.Search("Index").Data(), mod.Search("Name").Data()))
		log.Debug(fmt.Sprintf("\timage %v:%v", mod.Search("ImageName").Data(), mod.Search("Tag").Data()))
		log.Debug("\toptions:")
		for _, opt := range mod.Search("options").Children() {
			log.Debug(fmt.Sprintf("\t\t %v = %v", opt.Search("opt").Data(), opt.Search("val").Data()))
		}
		log.Debug("\targuments:")
		for _, arg := range mod.Search("arguments").Children() {
			log.Debug(fmt.Sprintf("\t\t %-15v= %v", arg.Search("arg").Data(), arg.Search("val").Data()))
		}
	}
	return true
}

// Simple pretty printer for arbitrary json
// Depends on the gabs package
func PrettyPrintJson(jsonBytes []byte) {
	jsonObj, err := gabs.ParseJSON(jsonBytes)
	if err != nil {
		panic(err)
	}
	fmt.Println(jsonObj.StringIndent("", "  "))
}
