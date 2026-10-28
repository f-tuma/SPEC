package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type installerFixture struct{ root, packages, bin, tools, asset, marker string }

func binaryInstallerFixture(t *testing.T) installerFixture {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("POSIX installer; Windows uses scripts/test-install.ps1")
	}
	root := t.TempDir()
	f := installerFixture{root: root, packages: filepath.Join(root, "release packages"), bin: filepath.Join(root, "bin with spaces"), tools: filepath.Join(root, "tools"), marker: filepath.Join(root, "executed")}
	f.asset = "SPEC_0.3.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	for _, dir := range []string{f.packages, f.bin, f.tools} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	f.packageArchive(t, "0.3.0", false)
	f.stub(t, "go", "echo 'installer must not compile' >&2; exit 99")
	return f
}
func shellArg(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
func (f installerFixture) stub(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.tools, name), []byte("#!/bin/sh\nset -eu\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
}
func (f installerFixture) packageArchive(t *testing.T, version string, extra bool) {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	files := map[string]string{"spec": "#!/bin/sh\nprintf x >> \"$SPEC_TEST_MARKER\"\nprintf '%s\\n' '" + version + "'\n", "LICENSE": "fixture license", "THIRD_PARTY_NOTICES.md": "fixture notices", "README.md": "fixture documentation"}
	if extra {
		files["../escaped"] = "must not be extracted"
	}
	for name, data := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(data))
	}
	tw.Close()
	gz.Close()
	if err := os.WriteFile(filepath.Join(f.packages, f.asset), buffer.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	f.checksum(t, fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes())), false)
}
func (f installerFixture) checksum(t *testing.T, digest string, duplicate bool) {
	t.Helper()
	line := digest + "  " + f.asset + "\n"
	if duplicate {
		line += line
	}
	if err := os.WriteFile(filepath.Join(f.packages, "SHA256SUMS"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
}
func (f installerFixture) run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	script, err := filepath.Abs("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", append([]string{script, "--bin-dir", f.bin}, args...)...)
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "TMPDIR="+f.root, "PATH="+f.tools+string(os.PathListSeparator)+os.Getenv("PATH"), "SPEC_TEST_MARKER="+f.marker, "SPEC_TEST_PACKAGES="+f.packages, "SPEC_TEST_ASSET="+f.asset, "SPEC_TEST_REQUESTS="+filepath.Join(f.root, "requests"))
	data, err := cmd.CombinedOutput()
	return string(data), err
}
func (f installerFixture) oldInstall(t *testing.T) {
	t.Helper()
	for _, name := range []string{"spec", "spec.LICENSE", "spec.THIRD_PARTY_NOTICES.md"} {
		if err := os.WriteFile(filepath.Join(f.bin, name), []byte("previous "+name), 0755); err != nil {
			t.Fatal(err)
		}
	}
}
func (f installerFixture) assertOld(t *testing.T) {
	t.Helper()
	for _, name := range []string{"spec", "spec.LICENSE", "spec.THIRD_PARTY_NOTICES.md"} {
		data, _ := os.ReadFile(filepath.Join(f.bin, name))
		if string(data) != "previous "+name {
			t.Fatalf("previous install changed: %s: %q", name, data)
		}
	}
}

func TestPrebuiltInstallerLocalAutoVersionWithoutCompiler(t *testing.T) {
	f := binaryInstallerFixture(t)
	out, err := f.run(t, "--from-dir", f.packages)
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	for _, name := range []string{"spec", "spec.LICENSE", "spec.THIRD_PARTY_NOTICES.md"} {
		if _, err := os.Stat(filepath.Join(f.bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(f.marker)
	if string(data) != "x" {
		t.Fatal("expected only the verified binary's version probe")
	}
	if _, err := os.Stat(filepath.Join(f.bin, ".spec-install.lock")); !os.IsNotExist(err) {
		t.Fatal("lock leaked")
	}
}
func TestPrebuiltInstallerRejectsBeforeReplacingExistingFiles(t *testing.T) {
	for _, kind := range []string{"checksum", "duplicate-checksum", "layout", "version", "multiple-versions"} {
		t.Run(kind, func(t *testing.T) {
			f := binaryInstallerFixture(t)
			f.oldInstall(t)
			switch kind {
			case "checksum":
				f.checksum(t, strings.Repeat("0", 64), false)
			case "duplicate-checksum":
				data, _ := os.ReadFile(filepath.Join(f.packages, f.asset))
				f.checksum(t, fmt.Sprintf("%x", sha256.Sum256(data)), true)
			case "layout":
				f.packageArchive(t, "0.3.0", true)
			case "version":
				f.packageArchive(t, "0.2.0", false)
			case "multiple-versions":
				other := strings.Replace(f.asset, "0.3.0", "0.4.0", 1)
				if err := os.WriteFile(filepath.Join(f.packages, other), []byte("unused"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := f.run(t, "--from-dir", f.packages); err == nil {
				t.Fatalf("expected refusal: %s", out)
			}
			f.assertOld(t)
			if kind != "version" {
				if _, err := os.Stat(f.marker); !os.IsNotExist(err) {
					t.Fatal("unverified payload executed")
				}
			}
			if _, err := os.Stat(filepath.Join(f.root, "escaped")); !os.IsNotExist(err) {
				t.Fatal("archive path traversal")
			}
		})
	}
}
func TestPrebuiltInstallerRecoversLateRenameFailure(t *testing.T) {
	f := binaryInstallerFixture(t)
	f.oldInstall(t)
	mv, err := exec.LookPath("mv")
	if err != nil {
		t.Fatal(err)
	}
	f.stub(t, "mv", "case \"$3\" in */new/spec) exit 1 ;; esac\nexec "+shellArg(mv)+" \"$@\"")
	if out, err := f.run(t, "--from-dir", f.packages); err == nil {
		t.Fatal("injected rename failure ignored:", out)
	}
	f.assertOld(t)
	if _, err := os.Stat(filepath.Join(f.bin, ".spec-install.lock")); !os.IsNotExist(err) {
		t.Fatal("lock leaked after failure")
	}
}
func TestPrebuiltInstallerPreservesSymlinkAndActiveLock(t *testing.T) {
	for _, kind := range []string{"symlink", "lock"} {
		t.Run(kind, func(t *testing.T) {
			f := binaryInstallerFixture(t)
			if kind == "lock" {
				os.Mkdir(filepath.Join(f.bin, ".spec-install.lock"), 0755)
			} else {
				target := filepath.Join(f.root, "owner-file")
				os.WriteFile(target, []byte("owner bytes"), 0644)
				if err := os.Symlink(target, filepath.Join(f.bin, "spec")); err != nil {
					t.Skip(err)
				}
			}
			if out, err := f.run(t, "--from-dir", f.packages); err == nil {
				t.Fatal("expected refusal:", out)
			}
			if kind == "lock" {
				if _, err := os.Stat(filepath.Join(f.bin, ".spec-install.lock")); err != nil {
					t.Fatal("removed another install's lock")
				}
			} else {
				data, _ := os.ReadFile(filepath.Join(f.root, "owner-file"))
				if string(data) != "owner bytes" {
					t.Fatal("symlink target changed")
				}
			}
		})
	}
}
func TestPrebuiltInstallerDownloadsWithPublicHTTPS(t *testing.T) {
	for _, pinned := range []bool{false, true} {
		t.Run(fmt.Sprint(pinned), func(t *testing.T) {
			f := binaryInstallerFixture(t)
			f.stub(t, "curl", `printf '%s\n' "$*" >> "$SPEC_TEST_REQUESTS"
destination=
while [ "$#" -gt 0 ]; do
  if [ "$1" = --output ]; then destination=$2; shift 2; else url=$1; shift; fi
done
case "$url" in
  */releases/latest) printf '{"tag_name":"v0.3.0"}\n' > "$destination" ;;
  */SHA256SUMS) cp "$SPEC_TEST_PACKAGES/SHA256SUMS" "$destination" ;;
  */"$SPEC_TEST_ASSET") cp "$SPEC_TEST_PACKAGES/$SPEC_TEST_ASSET" "$destination" ;;
  *) exit 22 ;;
esac`)
			args := []string{"--transport", "https"}
			if pinned {
				args = append(args, "--version", "v0.3.0")
			}
			out, err := f.run(t, args...)
			if err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			requests, _ := os.ReadFile(filepath.Join(f.root, "requests"))
			if strings.Contains(string(requests), "/releases/latest") == pinned {
				t.Fatal("wrong latest/pinned resolution", string(requests))
			}
			if !strings.Contains(string(requests), "--proto =https") {
				t.Fatal("HTTPS protocol guard missing")
			}
		})
	}
}
func TestPrebuiltInstallerUsesAuthenticatedGithubCLI(t *testing.T) {
	f := binaryInstallerFixture(t)
	f.stub(t, "curl", "echo 'private download must use gh' >&2; exit 88")
	f.stub(t, "gh", `printf '%s\n' "$*" >> "$SPEC_TEST_REQUESTS"
case "$1 $2" in
  'auth status') exit 0 ;;
  'release view') printf 'v0.3.0\n' ;;
  'release download')
    while [ "$#" -gt 0 ]; do if [ "$1" = --dir ]; then destination=$2; shift 2; else shift; fi; done
    cp "$SPEC_TEST_PACKAGES/$SPEC_TEST_ASSET" "$destination/$SPEC_TEST_ASSET"
    cp "$SPEC_TEST_PACKAGES/SHA256SUMS" "$destination/SHA256SUMS" ;;
  *) exit 2 ;;
esac`)
	if out, err := f.run(t); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	requests, _ := os.ReadFile(filepath.Join(f.root, "requests"))
	if !strings.Contains(string(requests), "--repo f-tuma/SPEC") || !strings.Contains(string(requests), "--pattern SHA256SUMS") {
		t.Fatal("missing repository/checksum selection")
	}
}

func TestSourceBuildWrapperKeepsCallerDirectory(t *testing.T) {
	f := binaryInstallerFixture(t)
	source := filepath.Join(f.root, "source")
	if err := os.MkdirAll(filepath.Join(source, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("scripts/install-from-source.sh")
	if err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(source, "scripts/install-from-source.sh")
	if err = os.WriteFile(wrapper, data, 0755); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(f.root, "source-test-cli")
	if err = os.WriteFile(helper, []byte("#!/bin/sh\npwd\nprintf '%s\\n' \"$@\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	f.stub(t, "go", "mkdir -p bin\ncp "+shellArg(helper)+" bin/spec")
	cmd := exec.Command("sh", wrapper, "--project", ".", "--dry-run")
	cmd.Dir = f.root
	cmd.Env = append(os.Environ(), "PATH="+f.tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	// pwd prints the physical directory, as on macOS /private/var.
	root, _ := filepath.EvalSymlinks(f.root)
	if string(out) != root+"\ninstall\n--project\n.\n--dry-run\n" {
		t.Fatal("source wrapper changed caller-relative paths:", string(out))
	}
}
