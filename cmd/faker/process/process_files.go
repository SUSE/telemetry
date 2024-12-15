package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/template"
)

func main() {
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	baseDir := filepath.Join(currentDir, "cmd", "faker")

	// Read environment variables
	clientId := rand.Intn(1000000) // Random client ID as an integer
	serverURL := os.Getenv("SERVER_URL")
	telemetryPath := os.Getenv("TELEMETRY_PATH")
	fileCountStr := os.Getenv("FILE_COUNT")

	if serverURL == "" || telemetryPath == "" || fileCountStr == "" {
		log.Fatalf("Missing required environment variables")
	}

	fileCount, err := strconv.Atoi(fileCountStr)
	if err != nil {
		log.Fatalf("Invalid FILE_COUNT: %v", err)
	}

	// Load the template config
	templatePath := filepath.Join(baseDir, "config", "client_template.yaml")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatalf("Failed to read template file: %v", err)
	}

	// Parse the template
	tmpl, err := template.New("config").Parse(string(templateContent))
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Generate the client-specific config
	configData := map[string]interface{}{
		"SERVER_URL":  serverURL,
		"CUSTOMER_ID": clientId,
	}
	var configBuffer bytes.Buffer
	if err := tmpl.Execute(&configBuffer, configData); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	// Save the generated config file
	configPath := filepath.Join(baseDir, "config", fmt.Sprintf("client_%d.yaml", clientId))
	if err := os.WriteFile(configPath, configBuffer.Bytes(), 0644); err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}

	fmt.Printf("Generated config file: %s\n", configPath)

	// Run the faker command to generate telemetry data
	if err := os.Chdir(baseDir); err != nil {
		log.Fatalf("Failed to change directory to %s: %v", baseDir, err)
	}

	// Run the faker command to generate data
	fakerCmd := exec.Command("go", "run", ".", "--count", fileCountStr)
	fakerCmd.Stdout = os.Stdout
	fakerCmd.Stderr = os.Stderr
	fmt.Println("Generating fake telemetry data...")
	if err := fakerCmd.Run(); err != nil {
		log.Fatalf("Faker command failed: %v", err)
	}
	fmt.Println("Fake telemetry data generated successfully.")

	// Change to the `cmd/authenticator` directory
	authDir := filepath.Join(currentDir, "cmd", "authenticator")
	if err := os.Chdir(authDir); err != nil {
		log.Fatalf("Failed to change directory to %s: %v", authDir, err)
	}

	// Authenticate the client
	authCmd := exec.Command("go", "run", ".", "--config", configPath)
	authCmd.Stdout = os.Stdout
	authCmd.Stderr = os.Stderr
	fmt.Println("Authenticating client...")
	if err := authCmd.Run(); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Println("Client authenticated successfully.")

	// Change back to the telemetry generator directory
	genDir := filepath.Join(currentDir, "cmd", "generator")
	if err := os.Chdir(genDir); err != nil {
		log.Fatalf("Failed to change directory to %s: %v", genDir, err)
	}

	// Process telemetry files
	for i := 1; i < fileCount; i++ {
		file := filepath.Join(telemetryPath, fmt.Sprintf("telemetry_data_%d.json", i))
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("File %s does not exist. Skipping.\n", file)
			continue
		}

		cmd := exec.Command("go", "run", ".", "--config", configPath, "--telemetry=FAKER-GENERATED-DATA", "--tag=FAKERTEST", file)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to process file %s: %v", file, err)
		} else {
			fmt.Printf("Processed file: %s\n", file)
		}
	}

	fmt.Println("Telemetry submission completed.")
}
