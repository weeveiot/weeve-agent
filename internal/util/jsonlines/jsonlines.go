package jsonlines

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"os"
	"reflect"

	log "github.com/sirupsen/logrus"
)

func getOriginalSlice(ptrToSlice interface{}) (slice reflect.Value, err error) {
	ptr2sl := reflect.TypeOf(ptrToSlice)
	if ptr2sl.Kind() != reflect.Ptr {
		return reflect.ValueOf(nil), fmt.Errorf("expected pointer to slice, got %s", ptr2sl.Kind())
	}

	originalSlice := reflect.Indirect(reflect.ValueOf(ptrToSlice))
	sliceType := originalSlice.Type()
	if sliceType.Kind() != reflect.Slice {
		return reflect.ValueOf(nil), fmt.Errorf("expected pointer to slice, got pointer to %s", sliceType.Kind())
	}
	return originalSlice, nil
}

// Decode reads the next JSON Lines-encoded value that reads
// from r and stores it in the slice pointed to by ptrToSlice.
func Decode(r io.Reader, ptrToSlice interface{}) error {
	originalSlice, err := getOriginalSlice(ptrToSlice)
	if err != nil {
		return err
	}

	slElem := originalSlice.Type().Elem()
	//originalSlice := reflect.Indirect(reflect.ValueOf(ptrToSlice))
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		//create new object
		newObj := reflect.New(slElem).Interface()
		item := scanner.Bytes()
		err := json.Unmarshal(item, newObj)
		if err != nil {
			return err
		}
		ptrToNewObj := reflect.Indirect(reflect.ValueOf(newObj))
		originalSlice.Set(reflect.Append(originalSlice, ptrToNewObj))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Encode writes the JSON Lines encoding of ptrToSlice to the w stream
func Encode(w io.Writer, ptrToSlice interface{}) error {
	originalSlice, err := getOriginalSlice(ptrToSlice)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	for i := 0; i < originalSlice.Len(); i++ {
		elem := originalSlice.Index(i).Interface()
		err = enc.Encode(elem)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateIfNotExist(filePath string) error {
	_, err := os.Stat(filePath)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		log.Info("Creating new file ", filePath)
		_, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err == nil {
			log.Info("New file created ", filePath)
			return nil
		} else {
			log.Error("Unable to create file ", filePath, err)
		}
	}

	return err
}

func Read(jsonFile string, pkField string, pkVal string, filter map[string]string, excludeKey bool) ([]map[string]interface{}, error) {
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

		if pkField != "" && pkVal != "" {
			if lnVal[pkField] == pkVal && !excludeKey {
				val = append(val, lnVal)
			} else if lnVal[pkField] != pkVal && excludeKey {
				val = append(val, lnVal)
			}
		} else {
			if filter != nil {
				add := true
				for k, v := range filter {
					log.Debug(k, " value is ", v)
					if lnVal[k] != v {
						add = false
					}
				}
				if (add && !excludeKey) || (!add && excludeKey) {
					val = append(val, lnVal)
				}
			} else {
				val = append(val, lnVal)
			}
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

func Delete(jsonFile string, pkField string, pkVal string, filter map[string]string, excludeKey bool) bool {
	log.Debug("jsonlines >> Delete()")
	allExceptPk, err := Read(jsonFile, pkField, pkVal, filter, excludeKey)
	if err != nil {
		log.Error(err)
		CreateIfNotExist(jsonFile)

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
