package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/spf13/cobra"
)

// installMapFlag adds --map, which prints the command tree below the
// selected command instead of running it.
func installMapFlag(root *cobra.Command) {
	flags := root.PersistentFlags()
	flags.String("map", "", "Print the commands below this one instead of running it: tree (default), paths or json")
	flags.Lookup("map").NoOptDefVal = "tree"
	_ = flags.SetAnnotation("map", "speakeasy:group", []string{"Diagnostics"})
	_ = flags.SetAnnotation("map", flagutil.AnnotationDocSurface, []string{"true"})
	walkCommands(root, func(cmd *cobra.Command) {
		if cmd.RunE == nil {
			return
		}
		run := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if !flagutil.FlagChanged(cmd, "map") {
				return run(cmd, args)
			}
			format, _ := flagutil.GetStringFlag(cmd, "map")
			return writeCommandMap(cmd.OutOrStdout(), cmd, format)
		}
	})
}

type commandNode struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Usage    string         `json:"usage"`
	Summary  string         `json:"summary,omitempty"`
	Aliases  []string       `json:"aliases,omitempty"`
	Commands []*commandNode `json:"commands,omitempty"`
}

func buildCommandNode(cmd *cobra.Command) *commandNode {
	node := &commandNode{
		Name:    cmd.Name(),
		Path:    cmd.CommandPath(),
		Usage:   strings.Join(append([]string{cmd.CommandPath()}, strings.Fields(cmd.Use)[1:]...), " "),
		Summary: cmd.Short,
		Aliases: cmd.Aliases,
	}
	for _, child := range cmd.Commands() {
		if child.IsAvailableCommand() {
			node.Commands = append(node.Commands, buildCommandNode(child))
		}
	}
	return node
}

func writeCommandMap(w io.Writer, cmd *cobra.Command, format string) error {
	node := buildCommandNode(cmd)
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(node)
	case "paths":
		var walk func(*commandNode)
		walk = func(n *commandNode) {
			if len(n.Commands) == 0 {
				fmt.Fprintln(w, n.Usage)
			}
			for _, child := range n.Commands {
				walk(child)
			}
		}
		walk(node)
		return nil
	case "tree":
		fmt.Fprintln(w, node.Path)
		var walk func(*commandNode, string)
		walk = func(n *commandNode, indent string) {
			width := 0
			for _, child := range n.Commands {
				width = max(width, len(child.Name))
			}
			for _, child := range n.Commands {
				fmt.Fprintf(w, "%s%-*s  %s\n", indent, width, child.Name, child.Summary)
				walk(child, indent+"  ")
			}
		}
		walk(node, "  ")
		return nil
	}
	return flagutil.WithCLIValidation(fmt.Errorf("invalid --map %q: use tree, paths or json", format))
}
