package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brianvoe/gofakeit/v6"
)

type TelemetryData struct {
	Version      int    `json:"version"`
	Hostname     string `json:"hostname"`
	DistroTarget string `json:"distroTarget"`
	HwInfo       HwInfo `json:"hwInfo"`
}

type HwInfo struct {
	Hostname      string `json:"hostname"`
	CPUs          int    `json:"cpus"`
	Sockets       int    `json:"sockets"`
	Hypervisor    string `json:"hypervisor"`
	Arch          string `json:"arch"`
	UUID          string `json:"uuid"`
	CloudProvider string `json:"cloudProvider"`
	MemTotal      int    `json:"memTotal"`
}

func GenerateHostName() string {
	return fmt.Sprintf("%s-%d", gofakeit.AppName(), gofakeit.Number(1, 9999))
}

func GenerateFakeData() TelemetryData {
	hostname := GenerateHostName()
	return TelemetryData{
		Version:      gofakeit.Number(1, 10),
		Hostname:     hostname,
		DistroTarget: fmt.Sprintf("sle-%d-x86_64", gofakeit.Number(12, 15)),
		HwInfo: HwInfo{
			Hostname:      hostname,
			CPUs:          gofakeit.Number(1, 64),
			Sockets:       gofakeit.Number(1, 4),
			Hypervisor:    gofakeit.RandomString([]string{"KVM", "VMware", "Hyper-V", "Xen", ""}),
			Arch:          gofakeit.RandomString([]string{"amd64", "arm64", "arm"}),
			UUID:          gofakeit.UUID(),
			CloudProvider: gofakeit.RandomString([]string{"AWS", "Azure", "GCP", "On-Premise", ""}),
			MemTotal:      gofakeit.Number(1024, 65536),
		},
	}
}

func SaveToJSONFile(data TelemetryData, dir string, index int) (string, error) {
	fileName := fmt.Sprintf("telemetry_data_%d.json", index)
	filePath := filepath.Join(dir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return "", fmt.Errorf("failed to encode JSON: %w", err)
	}

	return filePath, nil
}

func main() {
	outputDir := flag.String("output", "fake_telemetry_data", "Ouput Directory")
	numEntries := flag.Int("count", 30, "Number of fake telemetry entries to generate")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, os.ModePerm); err != nil {
		fmt.Printf("Failed to create output directory: %s\n", err)
		os.Exit(1)
	}

	for i := 1; i <= *numEntries; i++ {
		data := GenerateFakeData()
		filePath, err := SaveToJSONFile(data, *outputDir, i)
		if err != nil {
			fmt.Printf("Error saving file: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated file: %s\n", filePath)
	}

	fmt.Printf("Generated %d fake telemetry data files in directory: %s\n", *numEntries, *outputDir)
}
