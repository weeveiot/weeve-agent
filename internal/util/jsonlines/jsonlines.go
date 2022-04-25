package jsonlines

import (
	"bufio"
	"encoding/json"

	"os"

	log "github.com/sirupsen/logrus"
)

func Read(jsonFile string, filter map[string]string, excludeKey bool) ([]map[string]interface{}, error) {
	var val []map[string]interface{}
	file, err := os.Open(jsonFile)
	if err != nil {
		return val, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanLines)
	var text []string

	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	for _, each_ln := range text {
		log.Debug(each_ln)
		var lnVal map[string]interface{}
		json.Unmarshal([]byte(each_ln), &lnVal)

		if filter != nil {
			add := true
			for k, v := range filter {
				log.Debug(k, " value is ", v)
				if lnVal[k] != v {
					add = false
				}
			}
			if add != excludeKey {
				val = append(val, lnVal)
			}
		} else {
			val = append(val, lnVal)
		}
	}

	return val, err
}

func Insert(jsonFile string, jsonString string) bool {
	f, err := os.OpenFile(jsonFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return false
	}
	defer f.Close()
	if _, err := f.WriteString(jsonString + "\n"); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func Delete(jsonFile string, filter map[string]string, excludeKey bool) bool {
	log.Debug("jsonlines >> Delete()")
	allExceptPk, err := Read(jsonFile, filter, excludeKey)
	if err != nil {
		return false
	}

	err = os.Remove(jsonFile)
	if err != nil {
		return false
	}

	log.Debug("File Deleted")
	f, err := os.OpenFile(jsonFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err)
		return false
	}
	defer f.Close()

	for _, rec := range allExceptPk {
		jsonString, err := json.Marshal(rec)
		if _, err1 := f.WriteString(string(jsonString) + "\n"); err != nil {
			log.Error(err1)
		}
	}
	return true
}
