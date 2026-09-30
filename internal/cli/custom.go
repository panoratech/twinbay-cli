package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/panoratech/twinbay-cli/internal/cli/environmentexports"
	"github.com/panoratech/twinbay-cli/internal/cli/environmentlogs"
	"github.com/panoratech/twinbay-cli/internal/cli/tests"
	"github.com/panoratech/twinbay-cli/internal/cli/twinrecords"
	"github.com/panoratech/twinbay-cli/internal/client"
	"github.com/panoratech/twinbay-cli/internal/flagutil"
	"github.com/panoratech/twinbay-cli/internal/interactive"
	"github.com/panoratech/twinbay-cli/internal/output"
	"github.com/panoratech/twinbay-cli/internal/usage"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// installCustomCommands applies workflow-level behavior the generator cannot
// express. NewRootCommand calls it once, before the interactive and usage
// interceptors wrap the tree, so everything here is intercepted like
// generated code.
func installCustomCommands(root *cobra.Command) error {
	root.SetGlobalNormalizationFunc(normalizeFlagName)
	aliasAPIKeyEnv(root)
	if err := nestSubresources(root); err != nil {
		return err
	}
	initRawCmds(root)
	initConfigCmd(root)
	consolidateWhoami(root)
	walkCommands(root, func(cmd *cobra.Command) {
		// create_export's nullable body surfaces twice: --body and --body-param.
		if cmd.Flags().Lookup("body") != nil && cmd.Flags().Lookup("body-param") != nil {
			_ = cmd.Flags().MarkHidden("body-param")
		}
		if (cmd.Name() == "delete" || cmd.Name() == "revoke") && cmd.RunE != nil {
			requireConfirmation(cmd)
		}
	})
	installMapFlag(root)
	return nil
}

func walkCommands(cmd *cobra.Command, visit func(*cobra.Command)) {
	visit(cmd)
	for _, child := range cmd.Commands() {
		walkCommands(child, visit)
	}
}

// flagAliases maps accepted spellings to generated flag names.
var flagAliases = map[string]string{
	"api-key": "organization-api-key",
	// "version" is reserved by the generator; the path parameter is named version.
	"version-param": "version",
}

func normalizeFlagName(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	if canonical, ok := flagAliases[name]; ok {
		return pflag.NormalizedName(canonical)
	}
	return pflag.NormalizedName(name)
}

func aliasAPIKeyEnv(root *cobra.Command) {
	if value := os.Getenv("TWINBAY_API_KEY"); value != "" && os.Getenv("CLI_TWINBAY_ORGANIZATION_API_KEY") == "" {
		_ = os.Setenv("CLI_TWINBAY_ORGANIZATION_API_KEY", value)
	}
	if f := root.PersistentFlags().Lookup("organization-api-key"); f != nil {
		f.Usage += " (alias --api-key; env TWINBAY_API_KEY)"
	}
}

// nestedPositionals lists path parameters, in URL order, that nested
// commands accept as arguments. Single-parameter commands already accept
// theirs through the generated positional flag.
var nestedPositionals = map[string][]string{
	"environments logs retrieve":    {"environment-id", "request-id"},
	"environments exports retrieve": {"environment-id", "export-id"},
	"environments exports download": {"environment-id", "export-id"},
	"tests versions retrieve":       {"test-id", "version"},
	"twins records list":            {"twin-id", "resource"},
	"twins records update":          {"twin-id", "resource", "external-id"},
}

// nestSubresources re-parents generated subresource groups under their
// owning resource. The old top-level paths keep working, hidden and deprecated.
func nestSubresources(root *cobra.Command) error {
	for _, nest := range []struct {
		parent, old, name, short string
		init                     func(*cobra.Command) error
	}{
		{"environments", "environment-logs", "logs", "Requests an environment's twins served", environmentlogs.InitEnvironmentLogsRoot},
		{"environments", "environment-exports", "exports", "Exports of an environment's traffic", environmentexports.InitEnvironmentExportsRoot},
		{"twins", "twin-records", "records", "Records a provisioned twin holds", twinrecords.InitTwinRecordsRoot},
	} {
		parent := commandAt(root, []string{nest.parent})
		if parent == nil {
			return fmt.Errorf("nest %s: parent %q does not exist", nest.old, nest.parent)
		}
		if err := nest.init(parent); err != nil {
			return fmt.Errorf("nest %s: %w", nest.old, err)
		}
		group := commandAt(parent, []string{nest.old})
		if group == nil {
			return fmt.Errorf("nest %s: generated group was renamed", nest.old)
		}
		group.Use, group.Short, group.Long, group.Aliases = nest.name, nest.short, nest.short, nil
		for _, leaf := range group.Commands() {
			adoptNestedCommand(leaf, nest.old+" "+leaf.Name())
		}
		deprecateTree(commandAt(root, []string{nest.old}), root.Name()+" "+nest.parent+" "+nest.name)
	}

	// Test versions are operations on the tests group itself, so build a
	// fresh tests tree and adopt its version commands.
	testsCmd := commandAt(root, []string{"tests"})
	scratch := &cobra.Command{Use: "scratch"}
	if testsCmd == nil || tests.InitTestsRoot(scratch) != nil {
		return errors.New("nest tests versions: tests group unavailable")
	}
	versions := &cobra.Command{
		Use:         "versions",
		Short:       "Immutable versions of a test",
		Long:        "Every update to a test creates a new immutable version; evaluators compile a specific one.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"speakeasy_cli_group": "true"},
		RunE:        func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	usage.MarkDynamic(versions)
	testsCmd.AddCommand(versions)
	fresh := commandAt(scratch, []string{"tests"})
	for _, move := range []struct{ old, name string }{{"list-versions", "list"}, {"retrieve-version", "retrieve"}} {
		leaf := commandAt(fresh, []string{move.old})
		if leaf == nil {
			return fmt.Errorf("nest tests versions: generated %q was renamed", move.old)
		}
		fresh.RemoveCommand(leaf)
		leaf.Use, leaf.Aliases = move.name+strings.TrimPrefix(leaf.Use, move.old), nil
		versions.AddCommand(leaf)
		adoptNestedCommand(leaf, "tests "+move.old)
		deprecateTree(commandAt(testsCmd, []string{move.old}), root.Name()+" tests versions "+move.name)
	}
	return nil
}

// adoptNestedCommand points examples at the nested path and accepts path
// parameters as arguments. oldPath is the command path under the root
// before nesting, e.g. "environment-logs list".
func adoptNestedCommand(cmd *cobra.Command, oldPath string) {
	newPath := strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
	names := nestedPositionals[newPath]
	if names != nil {
		cmd.Use = cmd.Name()
		for _, name := range names {
			cmd.Use += " [" + name + "]"
			if f := cmd.Flags().Lookup(name); f != nil {
				f.Usage += " (or pass it as the [" + name + "] argument)"
			}
		}
		cmd.Args = positionalFlagArgs(names)
	} else if name := cmd.Annotations[flagutil.AnnotationPositionalFlag]; name != "" {
		names = []string{name}
	}

	lines := strings.Split(cmd.Example, "\n")
	for i, line := range lines {
		if !strings.Contains(line, " "+oldPath) {
			continue
		}
		line = strings.Replace(line, " "+oldPath, " "+newPath, 1)
		var values []string
		for _, name := range names {
			re := regexp.MustCompile(` --` + regexp.QuoteMeta(name) + `(?:-param)?[ =](\S+)`)
			if match := re.FindStringSubmatch(line); match != nil {
				values = append(values, match[1])
				line = strings.Replace(line, match[0], "", 1)
			}
		}
		if len(values) > 0 {
			line = strings.Replace(line, " "+newPath, " "+newPath+" "+strings.Join(values, " "), 1)
		}
		lines[i] = line
	}
	cmd.Example = strings.Join(lines, "\n")
}

// positionalFlagArgs assigns arguments to flags in order.
func positionalFlagArgs(names []string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > len(names) {
			return fmt.Errorf("accepts at most %d arg(s), received %d", len(names), len(args))
		}
		for i, value := range args {
			if flagutil.FlagChanged(cmd, names[i]) {
				return fmt.Errorf("pass %s once: as the [%s] argument or via --%s, not both", names[i], names[i], names[i])
			}
			if err := cmd.Flags().Set(names[i], value); err != nil {
				return err
			}
		}
		return nil
	}
}

func deprecateTree(cmd *cobra.Command, replacement string) {
	if cmd == nil {
		return
	}
	cmd.Hidden = true
	// Not cobra's Deprecated: it prints to stdout when one is set, and machine
	// mode keeps stderr silent on success.
	if run := cmd.RunE; run != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if !output.IsMachineMode(cmd) {
				fmt.Fprintf(cmd.ErrOrStderr(), "%q is deprecated; use %q instead.\n", cmd.CommandPath(), replacement)
			}
			return run(cmd, args)
		}
	}
	for _, child := range cmd.Commands() {
		deprecateTree(child, replacement+" "+child.Name())
	}
}

// requireConfirmation guards an irreversible command behind a prompt, or
// --confirm when no one can answer one.
func requireConfirmation(cmd *cobra.Command) {
	cmd.Flags().Bool("confirm", false, "Skip the confirmation prompt (required when not running interactively)")
	run := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if confirmed, _ := cmd.Flags().GetBool("confirm"); confirmed || client.IsDryRun(cmd) {
			return run(cmd, args)
		}
		if !interactive.Resolve(cmd).PromptRequiredInputs() {
			return flagutil.WithCLIValidation(fmt.Errorf("%s cannot be undone; pass --confirm to run it without a prompt", cmd.CommandPath()))
		}
		target := strings.Join(args, " ")
		if name := cmd.Annotations[flagutil.AnnotationPositionalFlag]; target == "" && name != "" {
			target, _ = flagutil.GetStringFlag(cmd, name)
		}
		confirmed := false
		if err := huh.NewConfirm().
			Title(strings.TrimSpace(cmd.CommandPath() + " " + target)).
			Description("This cannot be undone.").
			Affirmative("Yes").
			Negative("No").
			Value(&confirmed).
			Run(); err != nil {
			return fmt.Errorf("confirm: %w", err)
		}
		if !confirmed {
			return errors.New("cancelled; nothing was changed")
		}
		return run(cmd, args)
	}
}

func consolidateWhoami(root *cobra.Command) {
	top := commandAt(root, []string{"whoami"})
	auth := commandAt(root, []string{"auth", "whoami"})
	if top == nil || auth == nil {
		return
	}
	top.Short, top.Long, top.RunE = auth.Short, auth.Long+"\n\nSame as \"auth whoami\".", auth.RunE
}

// exactArgs is cobra.ExactArgs that lets --usage and --map through.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if usage.UsageRequested(cmd) || flagutil.FlagChanged(cmd, "map") {
			return nil
		}
		return cobra.ExactArgs(n)(cmd, args)
	}
}

// commandAt walks path from root by command name. Kept here rather than
// reusing the generated findCommandByPath, which only exists while the spec
// declares intent commands.
func commandAt(root *cobra.Command, path []string) *cobra.Command {
	for _, name := range path {
		var next *cobra.Command
		for _, child := range root.Commands() {
			if child.Name() == name {
				next = child
				break
			}
		}
		if next == nil {
			return nil
		}
		root = next
	}
	return root
}
