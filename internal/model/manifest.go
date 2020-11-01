package model

import "github.com/Jeffail/gabs/v2"

type Manifest struct {
	data []byte;
	manifest gabs.Container;
}

func (m Manifest) ParseJSON(data []byte) gabs.Container {
	jsonParsed, err := gabs.ParseJSON(data)
	if err != nil {
		panic(err)
	}
	return *jsonParsed
}