package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

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

		log.Debug(fmt.Sprintf("\t%-10v %v", methods[0], path))

		return nil
	})
}

func GetApi(host string) string {
	fmt.Println("Http Get...", host)
	resp, err := http.Get(host)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert response body to string
	bodyString := string(bodyBytes)
	fmt.Println("API Response as String:\n" + bodyString)

	return bodyString
}

func PostApi(nextHost string, jsonReq []byte) bool {
	fmt.Printf("Post host %s", nextHost)
	resp, err := http.Post(nextHost, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
	if err != nil {
		fmt.Println(err)
		return false
	}

	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	// Convert response body to string
	bodyString := string(bodyBytes)
	fmt.Println(bodyString)
	return true
}

// StringArrayContains tells whether a contains x.
func StringArrayContains(stringArray []string, findString string) bool {
	for _, n := range stringArray {
		if findString == n {
			return true
		}
	}
	return false
}

func GetExeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Could not get the path to the executable.")
	}
	dir := filepath.Dir(exePath)
	return dir
}
