package main

import (
	"fmt"

	"github.com/Jeffail/gabs/v2"
)



func main() {
	jsonParsed1, err := gabs.ParseJSON([]byte(`{"array":["first","second","third"]}`))
	if err != nil {
		panic(err)
	}

	for _, child := range jsonParsed1.S("array").Children() {
		fmt.Println(child.Data().(string))
	}

	fmt.Println("ARRAY OF STRINGS")
	jsonParsed2, err := gabs.ParseJSON([]byte(`["first","second","third"]`))

	if err != nil {
		panic(err)
	}

	for _, child := range jsonParsed2.Children() {
		fmt.Println(child.Data().(string))
	}

	jsonParsed3, err := gabs.ParseJSON([]byte(`[{"arg":"InBroker", "val":"localhost:1883"},
	{"arg":"ProcessName", "val":"container-1"},
	{"arg":"InTopic", "val":"topic/source"},
	{"arg":"InClient", "val":"weevenetwork/go-mqtt-gobot"},
	{"arg":"OutBroker", "val":"localhost:1883"},
	{"arg":"OutTopic", "val":"topic/c2"},
	{"arg":"OutClient", "val":"weevenetwork/go-mqtt-gobot"}]`))

	if err != nil {
		panic(err)
	}

	for _, child := range jsonParsed3.Children() {
		fmt.Println(child.Data().(string))
	}


}