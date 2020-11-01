package model

import "github.com/Jeffail/gabs/v2"

type Manifest struct {
	data []byte;
	manifest gabs.Container;
}

// Create a Manifest type
func ParseJSONManifest(data []byte) Manifest {
	var thisManifest = Manifest{}
	thisManifest.data = data
	jsonParsed, err := gabs.ParseJSON(thisManifest.data )
	if err != nil {
		panic(err)
	}

	thisManifest.manifest = *jsonParsed
	return thisManifest
}

// func (m Manifest) ImageNamesList []string {
// 	for _, mod := range m.manifest.Search("Modules").Children() {
// 		imageNamesList = append(imageNamesList, mod.Search("ImageName").Data().(string)+":"+mod.Search("Tag").Data().(string))
// 	}
// }

