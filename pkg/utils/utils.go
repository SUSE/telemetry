package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"

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

func CompressGZIP(data []byte) (compressedData []byte, err error) {
	var tmpBuffer bytes.Buffer

	encoder, err := gzip.NewWriterLevel(&tmpBuffer, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	defer encoder.Close()

	_, err = encoder.Write(data)
	if err != nil {
		return nil, err
	}

	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return tmpBuffer.Bytes(), nil
}

func DecompressGZIP(compressedData []byte) (decompressedData []byte, err error) {
	tmpBuffer := bytes.NewBuffer(compressedData)
	reader, err := gzip.NewReader(tmpBuffer)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decompressedData, err = io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if err := reader.Close(); err != nil {
		return nil, err
	}

	return decompressedData, nil
}
