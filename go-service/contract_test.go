package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	goServiceImage = "specmatic-go-service:latest"
	specmaticImage = "specmatic/specmatic"
	containerPort  = "8090/tcp"
	internalPort   = "8090"
	serviceAlias   = "go-service"

	healthCheckPath = "/actuator/mappings"
	apiSpecFile     = "products_api.yaml"
	startupTimeout  = 30 * time.Second
)

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestSpecmaticContract(t *testing.T) {
	ctx := context.Background()

	net, err := setupTestNetwork(ctx, t)
	if err != nil {
		t.Fatalf("Failed to setup test network: %v", err)
	}
	defer cleanupNetwork(ctx, net, t)

	container, err := startGoServiceContainer(ctx, t, net)
	if err != nil {
		t.Fatalf("Failed to start Go service container: %v", err)
	}
	defer cleanupContainer(ctx, container, t)

	if err := runSpecmaticTests(t, net); err != nil {
		t.Fatalf("Specmatic tests failed: %v", err)
	}
}

func setupTestNetwork(ctx context.Context, t *testing.T) (*tc.DockerNetwork, error) {
	t.Log("Creating test network...")

	net, err := network.New(ctx, network.WithDriver("bridge"))
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	t.Logf("Test network created successfully")
	return net, nil
}

func startGoServiceContainer(ctx context.Context, t *testing.T, net *tc.DockerNetwork) (tc.Container, error) {
	t.Log("Starting Go service container...")

	containerReq := tc.ContainerRequest{
		Image:        goServiceImage,
		ExposedPorts: []string{containerPort},
		Networks:     []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {serviceAlias},
		},
		WaitingFor: wait.ForHTTP(healthCheckPath).
			WithStartupTimeout(startupTimeout),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: containerReq,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	t.Logf("Go service container started successfully with alias '%s'", serviceAlias)
	return container, nil
}

func runSpecmaticTests(t *testing.T, net *tc.DockerNetwork) error {
	t.Log("Running Specmatic contract tests...")

	baseURL := fmt.Sprintf("http://%s:%s", serviceAlias, internalPort)
	projectDir, err := getProjectDir(t)
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	cmd := buildSpecmaticCommand(baseURL, projectDir, net.Name)

	t.Logf("Executing command: %s", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()

	t.Log("Specmatic output:")
	t.Log(string(output))

	if err != nil {
		return fmt.Errorf("specmatic execution failed: %w", err)
	}

	t.Log("Specmatic contract tests completed successfully")
	return nil
}

func buildSpecmaticCommand(baseURL, projectDir, networkName string) *exec.Cmd {
	oauthToken := getEnvWithDefault("SPECMATIC_OAUTH2_TOKEN", "")
	customResponse := getEnvWithDefault("CUSTOM_RESPONSE", "false")
	generativeTests := getEnvWithDefault("SPECMATIC_GENERATIVE_TESTS", "false")
	onlyPositive := getEnvWithDefault("ONLY_POSITIVE", "false")

	args := []string{
		"run", "--rm",
		"--network", networkName,
		"-e", fmt.Sprintf("SPECMATIC_OAUTH2_TOKEN=%s", oauthToken),
		"-e", fmt.Sprintf("CUSTOM_RESPONSE=%s", customResponse),
		"-e", fmt.Sprintf("SPECMATIC_GENERATIVE_TESTS=%s", generativeTests),
		"-e", fmt.Sprintf("ONLY_POSITIVE=%s", onlyPositive),
		"-v", fmt.Sprintf("%s:/app", projectDir),
		"-w", "/app",
		specmaticImage,
		"test", apiSpecFile,
		"--testBaseURL", baseURL,
	}

	return exec.Command("docker", args...)
}

func getProjectDir(t *testing.T) (string, error) {
	dir, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get project root: %w", err)
	}

	projectDir := strings.TrimSpace(string(dir))
	t.Logf("Project directory: %s", projectDir)

	return projectDir, nil
}

func cleanupContainer(ctx context.Context, container tc.Container, t *testing.T) {
	if container == nil {
		return
	}

	t.Log("Cleaning up Go service container...")
	if err := container.Terminate(ctx); err != nil {
		t.Errorf("Failed to terminate container: %v", err)
	} else {
		t.Log("Go service container terminated successfully")
	}
}

func cleanupNetwork(ctx context.Context, net *tc.DockerNetwork, t *testing.T) {
	if net == nil {
		return
	}

	t.Log("Cleaning up test network...")
	if err := net.Remove(ctx); err != nil {
		t.Errorf("Failed to remove network: %v", err)
	} else {
		t.Log("Test network removed successfully")
	}
}
