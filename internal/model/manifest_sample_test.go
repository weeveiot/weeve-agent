package model

import (
	"fmt"
	"testing"

	_ "gitlab.com/weeve/edge-server/edge-pipeline-service/testing"
)

func TestLoad(t *testing.T) {
	fmt.Println("Load the sample manifest")
	var sampleManifestBytesMVP []byte = LoadJsonBytes("manifest/mvp-manifest.json")
	// fmt.Println(sampleManifestBytesMVP)
	manifest, _ := ParseJSONManifest(sampleManifestBytesMVP)
	// fmt.Print(res.ContainerNamesList())
	ContainerConfigs := manifest.GetContainerStart("")
	// fmt.Print(ContainerConfig.MountConfigs)
	fmt.Println("Container details:")
	for i, ContainerConf := range ContainerConfigs {
		fmt.Println(i, ContainerConf)
	}
	// for i := 0; i < count; i++ {

	// }
	fmt.Print(ContainerConfigs[0].MountConfigs)
	// fmt.Print(res)
	// res.PrintManifest()
	// code := t.Run()

	// os.Exit(code)
}
