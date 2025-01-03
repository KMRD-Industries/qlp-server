package utils

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

type JsonParser struct {
}

func NewJsonParser() *JsonParser {
	return &JsonParser{}
}

func (j *JsonParser) ParseConfig(filePath string) (Config, error) {
	var config Config
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file: ", err)
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, errors.New("couldn't parse JSON")
	}
	return config, nil
}
