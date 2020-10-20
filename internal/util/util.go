package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func PrintEndpoints(r *mux.Router) {
	log.Debug("Available endpoints are registered, walking router tree:")
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}
		methods, err := route.GetMethods()
		if err != nil {
			return nil
		}

		log.Debug("\t", methods, "\t", path)

		return nil
	})
}

func readJson(jsonFileName string) {
	jsonFile, err := os.Open(jsonFileName)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened " + jsonFileName)
	defer jsonFile.Close()
}

func GetApi(host string) string { //}, result interface{}) interface{} { //}, params []string, respStruct interface{}) {
	fmt.Println("Http Get...%s", host)
	resp, err := http.Get(host)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert response body to string
	bodyString := string(bodyBytes)
	fmt.Println("API Response as String:\n" + bodyString)

	// json.Unmarshal(bodyBytes, &result)
	// fmt.Printf("API Response as struct %+v\n", result)

	return bodyString
}

func PostApi(nextHost string, jsonReq []byte) bool {
	fmt.Printf("Post host %s", nextHost)
	resp, err := http.Post(nextHost, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
	if err != nil {
		fmt.Println(err)
		// panic(err)
		return false
	}

	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert response body to string
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)
	return true
}
