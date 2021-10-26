// data_access
package dao

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadAllData(objectName string) map[string]string {
	fmt.Println("Hello World!")
	var dataMap map[string]string
	dataMap = make(map[string]string)
	return dataMap
}

func ReadSingleData(objectName string, id string) map[string]string {
	fmt.Println("Hello World!")
	var dataMap map[string]string
	dataMap = make(map[string]string)
	return dataMap
}

func SaveData(table string, dataMap map[string]interface{}) map[string]interface{} {
	fmt.Println("Save data to json", table)
	jsonStr, err := json.Marshal(dataMap)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(jsonStr))
	return dataMap
}

func EditData(id string, dataMap map[string]string) map[string]string {
	fmt.Println("Hello World!")
	dataMap["id"] = id
	return dataMap
}

func DeleteData(id string) bool {
	fmt.Println("Hello World!")
	return true
}

func readJson(jsonFileName string) {
	// Check if file exists
	if _, err := os.Stat(jsonFileName); err == nil {
		jsonFile, err := os.Open(jsonFileName)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Successfully Opened " + jsonFileName)
		defer jsonFile.Close()

	} else if os.IsNotExist(err) {
		return
	}
}
