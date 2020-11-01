package model

import "github.com/Jeffail/gabs/v2"

type Manifest struct {
	data []byte;
	Manifest gabs.Container;
	ID string;
}

// Create a Manifest type
func ParseJSONManifest(data []byte) Manifest {
	var thisManifest = Manifest{}
	thisManifest.data = data
	jsonParsed, err := gabs.ParseJSON(thisManifest.data )
	if err != nil {
		panic(err)
	}

	thisManifest.Manifest = *jsonParsed
	thisManifest.ID = thisManifest.Manifest.Search("ID").Data().(string)
	return thisManifest
}

func (m Manifest) ImageNamesList()  []string {
	var imageNamesList []string
	for _, mod := range m.Manifest.Search("Modules").Children() {
		imageNamesList = append(imageNamesList, mod.Search("ImageName").Data().(string)+":"+mod.Search("Tag").Data().(string))
	}
	return imageNamesList
}

