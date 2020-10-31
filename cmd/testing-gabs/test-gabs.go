package main

import (
	"fmt"

	"github.com/Jeffail/gabs/v2"
)



func main() {
// jsonParsed, err := gabs.ParseJSON([]byte(`{"array":["first","second","third"]}`))
// if err != nil {
// 	panic(err)
// }

// for _, child := range jsonParsed.S("array").Children() {
// 	fmt.Println(child.Data().(string))
// }


jsonParsed, err := gabs.ParseJSON([]byte(`["first","second","third"]`))

if err != nil {
	panic(err)
}

for _, child := range jsonParsed.Children() {
	fmt.Println(child.Data().(string))
}

}