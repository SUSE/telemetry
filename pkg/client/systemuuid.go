package client

import (
	"log/slog"
	"os"
	"strings"

	"github.com/SUSE/telemetry/pkg/utils"
)

const (
	// Path to Linux hardware UUID file under /sys
	LINUX_SYSTEM_UUID_PATH = "/sys/class/dmi/id/product_uuid"
)

// retrieve the system uuid
func getSystemUUID() string {
	// TODO: determine appropriate environment specific system UUID path
	sysuuidPath := LINUX_SYSTEM_UUID_PATH

	// if identified system UUID path doesn't exist, return empty string
	if !utils.CheckPathExists(sysuuidPath) {
		slog.Debug(
			"Unable to locate the Linux hardware UUID",
			slog.String("path", sysuuidPath),
		)
		return ""
	}

	// if identified system UUID path can't be read, return empty string
	// NOTE: retrieving the contents may require superuser privileges.
	uuid, err := os.ReadFile(sysuuidPath)
	if err != nil {
		slog.Debug(
			"unable to retrieve the system UUID - superuser privs may be required",
			slog.String("path", sysuuidPath),
			slog.String("err", err.Error()),
		)
		return ""
	}

	// return the retrieved system uuid
	return strings.TrimSpace(string(uuid))
}
