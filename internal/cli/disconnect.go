package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect <agent-id>",
	Short: "Disconnect an agent from Athyr",
	Long: `Disconnect a registered agent from the Athyr server.

Use this to clean up stale agent registrations when an agent
process was killed without graceful shutdown.

Supports both full UUIDs and short ID prefixes.

Example:
  athyr-agent disconnect abc12345-1234-5678-9abc-def012345678
  athyr-agent disconnect abc12345 --api http://localhost:8080`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIDInput := args[0]
		apiServer, _ := cmd.Flags().GetString("api")

		// Resolve short ID to full UUID if needed
		agentID, err := resolveAgentID(apiServer, agentIDInput)
		if err != nil {
			return err
		}

		// Build request
		reqBody := map[string]string{"agent_id": agentID}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		// Send disconnect request
		url := fmt.Sprintf("%s/v1/disconnect", apiServer)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Parse response
		var result struct {
			OK      bool   `json:"ok"`
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("invalid response: %s", string(respBody))
		}

		if result.Code != 0 {
			return fmt.Errorf("disconnect failed: %s", result.Message)
		}

		fmt.Printf("Disconnected agent %s\n", agentID)
		return nil
	},
}

func init() {
	disconnectCmd.Flags().String("api", "http://localhost:8080", "Athyr HTTP API server")
	rootCmd.AddCommand(disconnectCmd)
}

// resolveAgentID resolves a short ID prefix to a full agent UUID.
func resolveAgentID(apiServer, idInput string) (string, error) {
	// If it looks like a full UUID, use it directly
	if len(idInput) == 36 && strings.Count(idInput, "-") == 4 {
		return idInput, nil
	}

	// Fetch agent list and match by prefix
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/v1/agents", apiServer))
	if err != nil {
		return "", fmt.Errorf("failed to list agents: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Agents []struct {
			ID   string `json:"id"`
			Card struct {
				Name string `json:"name"`
			} `json:"card"`
		} `json:"agents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse agents: %w", err)
	}

	var matches []string
	for _, agent := range result.Agents {
		if strings.HasPrefix(agent.ID, idInput) || agent.Card.Name == idInput {
			matches = append(matches, agent.ID)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no agent found matching '%s'", idInput)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple agents match '%s', be more specific", idInput)
	}

	return matches[0], nil
}
