package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/stretchr/testify/require"
)

func TestDiscoverDirectoryDefaultsToSelfContainedExampleFixture(t *testing.T) {
	require.Equal(t, defaultExampleDir, discoverDirectory(nil))
}

func TestDiscoverDirectoryRespectsExplicitOverride(t *testing.T) {
	require.Equal(t, "custom-dir", discoverDirectory([]string{"--dir", "custom-dir"}))
	require.Equal(t, "other-dir", discoverDirectory([]string{"--dir=other-dir"}))
	require.Equal(t, "short-dir", discoverDirectory([]string{"-d", "short-dir"}))
}

func TestRegisterExampleSharedSectionsSupportsRegistrySharedFixture(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "testdata", "jsverbs-example", "registry-shared")

	registry, err := jsverbs.ScanDir(dir)
	require.NoError(t, err)
	require.NoError(t, registerExampleSharedSections(dir, registry))

	commandMap := mustExampleCommandMap(t, registry)

	rows := runExampleCommand(t, commandMap["issues list-issues"], map[string]map[string]interface{}{
		"default": {
			"repo": "go-go-golems/go-go-goja",
		},
		"filters": {
			"state":  "closed",
			"labels": []string{"bug", "docs"},
		},
	})
	require.Equal(t, "go-go-golems/go-go-goja", rows[0]["repo"])
	require.Equal(t, "closed", rows[0]["state"])
	require.EqualValues(t, 2, rows[0]["labelCount"])

	summaryRows := runExampleCommand(t, commandMap["summary summarize-filters"], map[string]map[string]interface{}{
		"filters": {
			"state":  "open",
			"labels": []string{"enhancement"},
		},
	})
	require.Equal(t, "open", summaryRows[0]["state"])
	require.EqualValues(t, 1, summaryRows[0]["labelCount"])
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

func mustExampleCommandMap(t *testing.T, registry *jsverbs.Registry) map[string]cmds.Command {
	t.Helper()
	commands, err := registry.Commands()
	require.NoError(t, err)
	ret := map[string]cmds.Command{}
	for i, command := range commands {
		ret[registry.Verbs()[i].FullPath()] = command
	}
	return ret
}

func runExampleCommand(t *testing.T, command cmds.Command, valuesBySection map[string]map[string]interface{}) []map[string]interface{} {
	t.Helper()
	parsedValues, err := runner.ParseCommandValues(command, runner.WithValuesForSections(valuesBySection))
	require.NoError(t, err)

	glazeCommand, ok := command.(cmds.GlazeCommand)
	require.True(t, ok)

	gp := &captureProcessor{}
	err = glazeCommand.RunIntoGlazeProcessor(context.Background(), parsedValues, gp)
	require.NoError(t, err)
	err = gp.Close(context.Background())
	require.NoError(t, err)

	rows := make([]map[string]interface{}, 0, len(gp.rows))
	for _, row := range gp.rows {
		rows = append(rows, rowToMap(row))
	}
	return rows
}

func rowToMap(row types.Row) map[string]interface{} {
	ret := map[string]interface{}{}
	for pair := row.Oldest(); pair != nil; pair = pair.Next() {
		ret[pair.Key] = pair.Value
	}
	return ret
}

type captureProcessor struct {
	rows []types.Row
}

func (c *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	c.rows = append(c.rows, row)
	return nil
}

func (c *captureProcessor) Close(context.Context) error {
	return nil
}
