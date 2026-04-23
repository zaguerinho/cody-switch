package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doc",
		Short: "Manage room documents (PROTOCOL, MANIFESTO, STATUS)",
	}
	cmd.AddCommand(newDocListCmd())
	cmd.AddCommand(newDocReadCmd())
	cmd.AddCommand(newDocUpdateCmd())
	return cmd
}

func newDocListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <room>",
		Short: "List documents in a room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := apiCall("GET", "/api/v1/rooms/"+args[0]+"/docs", nil)
			return handleResponse(resp, err, func(data any) {
				docs, ok := data.([]any)
				if !ok || len(docs) == 0 {
					fmt.Println("No documents")
					return
				}
				for _, d := range docs {
					m, _ := d.(map[string]any)
					name, _ := m["name"].(string)
					size, _ := m["size"].(float64)
					fmt.Printf("  %-20s %d bytes\n", name, int(size))
				}
			})
		},
	}
}

func newDocReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <room> <doc>",
		Short: "Read a document",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := apiCall("GET", "/api/v1/rooms/"+args[0]+"/docs/"+args[1], nil)
			return handleResponse(resp, err, func(data any) {
				m, _ := data.(map[string]any)
				content, _ := m["content"].(string)
				fmt.Print(content)
			})
		},
	}
}

func newDocUpdateCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "update <room> <doc> [content]",
		Short: "Update a document",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var content string
			if file != "" {
				data, err := os.ReadFile(file)
				if err != nil {
					return printError("read file: " + err.Error())
				}
				content = string(data)
			} else if len(args) > 2 {
				content = args[2]
			} else {
				return printError("provide content as argument or --file")
			}

			body := map[string]string{"content": content}
			resp, err := apiCall("PUT", "/api/v1/rooms/"+args[0]+"/docs/"+args[1], body)
			return handleResponse(resp, err, func(data any) {
				fmt.Printf("Updated %s in %q\n", args[1], args[0])
			})
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Read content from file")
	return cmd
}
