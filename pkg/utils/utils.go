package utils

import (
	"encoding/json"

	"github.com/xyproto/randomstring"
)

func GenerateRandomString(length int) string {
	return randomstring.HumanFriendlyString(length)
}

func SerializeMap(m map[string]interface{}) (string, error) {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func DeserializeMap(jsonStr string) (map[string]interface{}, error) {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
