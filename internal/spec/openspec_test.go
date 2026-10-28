package spec

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPnpmBackendIsPinnedPrivateAndReusable(t *testing.T) {
	m, _, cache := managerFixture(t)
	m.LookPath = func(name string) (string, error) {
		if name == "openspec" {
			return "", exec.ErrNotFound
		}
		if name != "node" && name != "pnpm" {
			t.Fatalf("unexpected executable lookup: %s", name)
		}
		return "/bin/" + name, nil
	}
	base := filepath.Join(cache, "openspec", OpenSpecVersion, "pnpm")
	entry := filepath.Join(base, "node_modules/@fission-ai/openspec/bin/openspec.js")
	installs := 0
	m.Run = func(ctx context.Context, cwd string, env []string, argv ...string) (string, error) {
		if len(argv) == 2 && argv[0] == "/bin/node" {
			if argv[1] == "--version" {
				return "v22.23.4", nil
			}
			if argv[1] != entry {
				t.Fatalf("wrong cached executable: %v", argv)
			}
		}
		if argv[0] == "/bin/pnpm" {
			installs++
			if cwd != base || argv[1] != "add" {
				t.Fatalf("installation escaped the private directory: %s %v", cwd, argv)
			}
			for _, flag := range []string{"--save-exact", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "@fission-ai/openspec@1.15.0"} {
				if !slices.Contains(argv, flag) {
					t.Fatalf("missing install constraint %s: %v", flag, argv)
				}
			}
			for _, pair := range [][2]string{{"--dir", base}, {"--store-dir", filepath.Join(cache, "pnpm-store")}} {
				index := slices.Index(argv, pair[0])
				if index < 0 || index+1 >= len(argv) || argv[index+1] != pair[1] {
					t.Fatalf("wrong private path for %s: %v", pair[0], argv)
				}
			}
			if slices.Contains(argv, "--global") || slices.Contains(argv, "-g") {
				t.Fatal("global pnpm installation")
			}
			put(t, entry, "cached OpenSpec entry")
			return "installed", nil
		}
		if !slices.Equal(argv, []string{"/bin/node", entry, "--version"}) {
			t.Fatalf("unexpected command: %v", argv)
		}
		return OpenSpecVersion, nil
	}
	for range 2 {
		argv, method, err := m.ensureCLI(context.Background(), cache, "pnpm", "")
		if err != nil || method != "pnpm" || !slices.Equal(argv, []string{"/bin/node", entry}) {
			t.Fatalf("%v %s %v", argv, method, err)
		}
	}
	if installs != 1 {
		t.Fatalf("cached backend installed %d times", installs)
	}
}

func TestPnpmBackendReportsFailuresWithoutFallback(t *testing.T) {
	for _, test := range []struct {
		name, missing, nodeVersion, cliVersion, want string
		installError                                 bool
	}{
		{name: "missing node", missing: "node", want: "pnpm installation requires Node.js"},
		{name: "missing pnpm", missing: "pnpm", want: "pnpm on PATH"},
		{name: "old node", nodeVersion: "v20.18.3", want: "Node.js >=20.19.0"},
		{name: "invalid node version", nodeVersion: "unknown", want: "Node.js >=20.19.0"},
		{name: "install failed", installError: true, want: "registry unavailable"},
		{name: "wrong OpenSpec", cliVersion: "1.14.0", want: "unexpected version"},
	} {
		t.Run(test.name, func(t *testing.T) {
			m, _, cache := managerFixture(t)
			m.LookPath = func(name string) (string, error) {
				if name == "openspec" || name == test.missing {
					return "", exec.ErrNotFound
				}
				if name != "node" && name != "pnpm" {
					t.Fatalf("unexpected fallback: %s", name)
				}
				return "/bin/" + name, nil
			}
			m.Run = func(ctx context.Context, cwd string, env []string, argv ...string) (string, error) {
				if argv[0] == "/bin/pnpm" {
					if test.installError {
						return "", errors.New("registry unavailable")
					}
					return "installed", nil
				}
				if len(argv) == 2 {
					if test.nodeVersion != "" {
						return test.nodeVersion, nil
					}
					return "v22.23.4", nil
				}
				return test.cliVersion, nil
			}
			_, _, err := m.ensureCLI(context.Background(), cache, "pnpm", "")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("wanted %q, got %v", test.want, err)
			}
			if test.cliVersion != "" {
				for _, detail := range []string{OpenSpecVersion, filepath.Join(cache, "openspec", OpenSpecVersion, "pnpm"), "--cache-dir"} {
					if !strings.Contains(err.Error(), detail) {
						t.Fatalf("version error lacks recovery detail %q: %v", detail, err)
					}
				}
			}
		})
	}
}
