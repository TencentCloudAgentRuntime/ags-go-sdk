package command

import (
	"testing"
)

func TestBuildProcessConfig_DefaultShellWrap(t *testing.T) {
	// When no config is provided, cmd should be wrapped through /bin/sh -c
	pc := buildProcessConfig("ls /", nil)

	if pc.Cmd != "/bin/sh" {
		t.Errorf("expected Cmd = /bin/sh, got %q", pc.Cmd)
	}
	expectedArgs := []string{"-c", "ls /"}
	if len(pc.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(pc.Args), pc.Args)
	}
	for i, v := range expectedArgs {
		if pc.Args[i] != v {
			t.Errorf("Args[%d] = %q, want %q", i, pc.Args[i], v)
		}
	}
}

func TestBuildProcessConfig_EmptyArgs_StillWrapsWithBash(t *testing.T) {
	// When config is provided but Args is empty, should still use /bin/sh -c
	pc := buildProcessConfig("echo hello", &ProcessConfig{})

	if pc.Cmd != "/bin/sh" {
		t.Errorf("expected Cmd = /bin/sh, got %q", pc.Cmd)
	}
	expectedArgs := []string{"-c", "echo hello"}
	if len(pc.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(pc.Args), pc.Args)
	}
	for i, v := range expectedArgs {
		if pc.Args[i] != v {
			t.Errorf("Args[%d] = %q, want %q", i, pc.Args[i], v)
		}
	}
}

func TestBuildProcessConfig_WithArgs_UsesCmdDirectly(t *testing.T) {
	// This is the fix scenario: when Args are provided, cmd should be used as
	// the executable directly, not wrapped through /bin/sh -c.
	// Previously this would produce: /bin/sh [-c bash -lc "ls /"] which hangs.
	pc := buildProcessConfig("bash", &ProcessConfig{
		Args: []string{"-lc", "ls /"},
	})

	if pc.Cmd != "bash" {
		t.Errorf("expected Cmd = bash, got %q", pc.Cmd)
	}
	expectedArgs := []string{"-lc", "ls /"}
	if len(pc.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(pc.Args), pc.Args)
	}
	for i, v := range expectedArgs {
		if pc.Args[i] != v {
			t.Errorf("Args[%d] = %q, want %q", i, pc.Args[i], v)
		}
	}
}

func TestBuildProcessConfig_WithArgs_EnvsAndCwd(t *testing.T) {
	// When Args are provided along with Envs and Cwd, all should be set correctly.
	cwd := "/tmp"
	pc := buildProcessConfig("node", &ProcessConfig{
		Args: []string{"index.js", "--port", "3000"},
		Envs: map[string]string{"NODE_ENV": "production"},
		Cwd:  &cwd,
	})

	if pc.Cmd != "node" {
		t.Errorf("expected Cmd = node, got %q", pc.Cmd)
	}
	expectedArgs := []string{"index.js", "--port", "3000"}
	if len(pc.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(pc.Args), pc.Args)
	}
	for i, v := range expectedArgs {
		if pc.Args[i] != v {
			t.Errorf("Args[%d] = %q, want %q", i, pc.Args[i], v)
		}
	}
	if pc.Envs == nil || pc.Envs["NODE_ENV"] != "production" {
		t.Errorf("unexpected Envs: %v", pc.Envs)
	}
	if pc.Cwd == nil || *pc.Cwd != "/tmp" {
		t.Errorf("unexpected Cwd: %v", pc.Cwd)
	}
}

func TestBuildProcessConfig_NoArgs_EnvsAndCwd(t *testing.T) {
	// Without Args, should wrap through bash and still carry Envs/Cwd.
	cwd := "/home/user"
	pc := buildProcessConfig("make build", &ProcessConfig{
		Envs: map[string]string{"CC": "gcc"},
		Cwd:  &cwd,
	})

	if pc.Cmd != "/bin/sh" {
		t.Errorf("expected Cmd = /bin/sh, got %q", pc.Cmd)
	}
	expectedArgs := []string{"-c", "make build"}
	if len(pc.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(pc.Args), pc.Args)
	}
	for i, v := range expectedArgs {
		if pc.Args[i] != v {
			t.Errorf("Args[%d] = %q, want %q", i, pc.Args[i], v)
		}
	}
	if pc.Envs == nil || pc.Envs["CC"] != "gcc" {
		t.Errorf("unexpected Envs: %v", pc.Envs)
	}
	if pc.Cwd == nil || *pc.Cwd != "/home/user" {
		t.Errorf("unexpected Cwd: %v", pc.Cwd)
	}
}
