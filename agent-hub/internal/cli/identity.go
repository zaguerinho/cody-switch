package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Identity represents persistent agent identity.
type Identity struct {
	Alias       string `json:"alias"`
	Description string `json:"description,omitempty"`
}

func newIdentityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity",
		Short: "Manage persistent agent identity",
	}
	cmd.AddCommand(newIdentitySetCmd())
	cmd.AddCommand(newIdentityShowCmd())
	cmd.AddCommand(newIdentityClearCmd())
	return cmd
}

func newIdentitySetCmd() *cobra.Command {
	var alias, description string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set agent identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			if alias == "" {
				return printError("--alias is required")
			}
			id := Identity{Alias: alias, Description: description}
			data, err := json.MarshalIndent(id, "", "  ")
			if err != nil {
				return printError("marshal identity: " + err.Error())
			}

			dir := baseDir()
			if err := os.MkdirAll(dir, 0755); err != nil {
				return printError("create directory: " + err.Error())
			}

			identPath := filepath.Join(dir, "identity.json")
			if err := os.WriteFile(identPath, data, 0644); err != nil {
				return printError("write identity: " + err.Error())
			}

			fmt.Printf("Identity set: %s\n", alias)
			if description != "" {
				fmt.Printf("Description: %s\n", description)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "Agent alias (required)")
	cmd.Flags().StringVar(&description, "description", "", "Agent description")
	return cmd
}

func newIdentityShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check project-local first
			if data, err := os.ReadFile(".agent-hub-alias"); err == nil {
				alias := string(data)
				fmt.Printf("Identity (project-local): %s\n", alias)
				return nil
			}

			// Check global identity
			identPath := filepath.Join(baseDir(), "identity.json")
			data, err := os.ReadFile(identPath)
			if err != nil {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>`")
			}
			var id Identity
			if err := json.Unmarshal(data, &id); err != nil {
				return printError("parse identity: " + err.Error())
			}
			fmt.Printf("Identity (global): %s\n", id.Alias)
			if id.Description != "" {
				fmt.Printf("Description: %s\n", id.Description)
			}
			return nil
		},
	}
}

func newIdentityClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			identPath := filepath.Join(baseDir(), "identity.json")
			if err := os.Remove(identPath); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No identity to clear")
					return nil
				}
				return printError("remove identity: " + err.Error())
			}
			fmt.Println("Identity cleared")
			return nil
		},
	}
}
