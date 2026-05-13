package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildRootCommand_HasSyncAndMigrate(t *testing.T) {
	root := buildRootCommand()
	if root == nil {
		t.Fatalf("expected non-nil root command")
	}

	if root.Use != "tg" {
		t.Fatalf("expected root use tg, got: %s", root.Use)
	}

	if cmd, _, err := root.Find([]string{"sync"}); err != nil || cmd == nil {
		t.Fatalf("expected sync command to exist")
	}
	if cmd, _, err := root.Find([]string{"migrate"}); err != nil || cmd == nil {
		t.Fatalf("expected migrate command to exist")
	}
}

func TestMigrateBackfillHelp_Executes(t *testing.T) {
	root := buildRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"migrate", "backfill", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected migrate backfill help execute without error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Backfill local archives from database") {
		t.Fatalf("expected help output to contain backfill description")
	}
}

func TestSyncHelp_Executes(t *testing.T) {
	root := buildRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"sync", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected sync help execute without error, got: %v", err)
	}
}

func TestBuildRootCommand_HasMigrateBackfillSubcommand(t *testing.T) {
	root := buildRootCommand()
	if cmd, _, err := root.Find([]string{"migrate", "backfill"}); err != nil || cmd == nil || cmd.Name() != "backfill" {
		t.Fatalf("expected migrate backfill subcommand to exist")
	}
}

func TestMigrateBackfill_ConfigRequired(t *testing.T) {
	root := buildRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"migrate", "backfill"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error when config flag is missing")
	}
	if !strings.Contains(err.Error(), "required flag") {
		t.Fatalf("expected required flag error, got: %v", err)
	}
}

func TestBuildRootCommand_HasMigrateMoveLegacySubcommand(t *testing.T) {
	root := buildRootCommand()
	if cmd, _, err := root.Find([]string{"migrate", "move-legacy"}); err != nil || cmd == nil || cmd.Name() != "move-legacy" {
		t.Fatalf("expected migrate move-legacy subcommand to exist")
	}
}

func TestMigrateMoveLegacyHelp_Executes(t *testing.T) {
	root := buildRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"migrate", "move-legacy", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected migrate move-legacy help execute without error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Move legacy root markdown files to pending-delete directory") {
		t.Fatalf("expected help output to contain move-legacy description")
	}
}

func TestMigrateMoveLegacy_ConfigRequired(t *testing.T) {
	root := buildRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"migrate", "move-legacy"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error when config flag is missing")
	}
	if !strings.Contains(err.Error(), "required flag") {
		t.Fatalf("expected required flag error, got: %v", err)
	}
}
