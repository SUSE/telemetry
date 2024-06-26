package telemetrylib

import (
	"fmt"
	"log"

	"github.com/SUSE/telemetry/pkg/utils"
)

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
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func ShouldCompress(data []byte) (resultData []byte, compressed bool, err error) {
	compressedData, err := utils.CompressGZIP(data)
	if err != nil {
		log.Fatal(err)
	}

	if len(data) <= len(compressedData) {
		return data, false, err
	}

	return compressedData, true, nil
}
