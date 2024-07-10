package utils

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
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

// TODO: Both CompressWhenNeeded and DecompressWhenNeeded should be methods in TelemetryData

// TODO: check if it's worth trying to compress the data prior to compressing it (e.g: using entropy algorithms)
// This would save some CPU usage client side
// TODO: have telemetry data type be passed in as a parameter to further check if we should compress data and which algorithm to use (e.g: deflate or gzip)
func CompressWhenNeeded(data []byte) (resultData []byte, compression *string, err error) {
	// 'compression' is inserted as a sql.NullString, hence it is returned as a nullable string
	var validStr string = "gzip"

	// check whether it's worth compressing
	const MIN_SIZE_DATA_COMPRESSION = 80
	if len(data) <= MIN_SIZE_DATA_COMPRESSION {
		return data, nil, nil
	}

	compressedData, err := CompressGZIP(data)
	if err != nil {
		return data, nil, err
	}

	if len(data) <= len(compressedData) {
		return data, nil, nil
	}

	return compressedData, &validStr, nil
}

// TODO: have telemetry data type be passed in as a parameter to further check if we should decompress data and which algorithm to use (e.g: deflate or gzip)
func DecompressWhenNeeded(data []byte, compression sql.NullString) (resultData []byte, err error) {
	if compression.Valid {
		resultData, err = DecompressGZIP(data)
		if err != nil {
			return data, err
		}
		return resultData, nil
	}
	return data, nil
}

func HumanReadableSize(data []byte) string {
	const unit = 1024
	size := len(data)
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.3f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
