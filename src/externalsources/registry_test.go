package externalsources

import (
	"LumenForge/src/config"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadMissingRegistryIsEmpty(t *testing.T) {
	registry, err := loadFile(filepath.Join(t.TempDir(), "missing.json"), filePolicy{})
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}
	if !registry.missing || len(registry.sources) != 0 || len(registry.ordered) != 0 {
		t.Fatalf("missing registry = %#v", registry)
	}
}

func TestLoadValidUserRegistry(t *testing.T) {
	path := writeRegistry(t, 0o600, []testEntry{{
		ID:         "gpu-temperature",
		Name:       "GPU Temperature",
		Executable: helperExecutable(t),
		Args:       []string{"-test.run=^TestExternalSourceHelperProcess$", "--", "value", "42.5"},
	}})
	registry, err := loadFile(path, policyForMode(config.ServiceModeUser))
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}
	if got := registry.ordered; len(got) != 1 || got[0] != (Info{ID: "gpu-temperature", Name: "GPU Temperature"}) {
		t.Fatalf("browser-safe entries = %#v", got)
	}
}

func TestLoadValidSystemRegistryPolicy(t *testing.T) {
	path := writeRegistry(t, 0o640, []testEntry{{
		ID:         "system-source",
		Name:       "System Source",
		Executable: helperExecutable(t),
		Args:       []string{},
	}})
	policy := policyForMode(config.ServiceModeSystem)
	if !policy.checkOwner || policy.expectedOwner != 0 || !policy.checkWriteMode {
		t.Fatalf("system policy = %#v", policy)
	}
	if os.Geteuid() != 0 {
		// A non-root test process cannot create the required root-owned fixture.
		// Retain every other system-mode check while exercising valid decoding
		// through the policy seam used only by this test.
		policy.expectedOwner = uint32(os.Geteuid())
		policy.executable.allowedOwners = []uint32{uint32(os.Geteuid())}
	}
	registry, err := loadFile(path, policy)
	if err != nil {
		t.Fatalf("system trust policy rejected valid registry: %v", err)
	}
	if len(registry.sources) != 1 {
		t.Fatalf("system registry sources = %d, want 1", len(registry.sources))
	}
}

func TestExecutablePolicyForMode(t *testing.T) {
	currentOwner := uint32(os.Geteuid())
	tests := []struct {
		name          string
		mode          config.ServiceMode
		allowedOwners []uint32
	}{
		{name: "system", mode: config.ServiceModeSystem, allowedOwners: []uint32{0}},
		{name: "user", mode: config.ServiceModeUser, allowedOwners: []uint32{currentOwner, 0}},
		{name: "development", mode: config.ServiceModeDevelopment, allowedOwners: []uint32{currentOwner, 0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := policyForMode(test.mode).executable
			if !policy.checkOwner || !policy.checkWriteMode {
				t.Fatalf("executable policy = %#v", policy)
			}
			if fmt.Sprint(policy.allowedOwners) != fmt.Sprint(test.allowedOwners) {
				t.Fatalf("allowed owners = %v, want %v", policy.allowedOwners, test.allowedOwners)
			}
		})
	}
}

func TestLoadExecutableOwnershipPolicy(t *testing.T) {
	t.Run("system accepts root owner", func(t *testing.T) {
		if os.Geteuid() != 0 {
			t.Skip("root-owned executable fixture requires root")
		}
		executable := copyExecutable(t, helperExecutable(t))
		path := writeRegistry(t, 0o600, []testEntry{{
			ID: "root-system", Name: "Root System", Executable: executable, Args: []string{},
		}})
		if _, err := loadFile(path, policyForMode(config.ServiceModeSystem)); err != nil {
			t.Fatalf("root-owned executable rejected in system mode: %v", err)
		}
	})

	t.Run("system rejects non-root owner", func(t *testing.T) {
		executable := copyExecutable(t, helperExecutable(t))
		if os.Geteuid() == 0 {
			if err := os.Chown(executable, 65534, -1); err != nil {
				t.Skipf("non-root ownership fixture unavailable: %v", err)
			}
		}
		path := writeRegistry(t, 0o600, []testEntry{{
			ID: "non-root-system", Name: "Non-root System", Executable: executable, Args: []string{},
		}})
		policy := policyForMode(config.ServiceModeSystem)
		if os.Geteuid() != 0 {
			policy.expectedOwner = uint32(os.Geteuid())
		}
		_, err := loadFile(path, policy)
		if !errors.Is(err, ErrRegistryInvalid) || !strings.Contains(err.Error(), "owner") {
			t.Fatalf("non-root executable load error = %v, want ownership rejection", err)
		}
	})

	for _, mode := range []config.ServiceMode{config.ServiceModeUser, config.ServiceModeDevelopment} {
		t.Run(string(mode)+" accepts current owner", func(t *testing.T) {
			executable := copyExecutable(t, helperExecutable(t))
			path := writeRegistry(t, 0o600, []testEntry{{
				ID: "current-owner", Name: "Current Owner", Executable: executable, Args: []string{},
			}})
			if _, err := loadFile(path, policyForMode(mode)); err != nil {
				t.Fatalf("current-user-owned executable rejected in %s mode: %v", mode, err)
			}
		})

		t.Run(string(mode)+" accepts root owner", func(t *testing.T) {
			if os.Geteuid() != 0 {
				t.Skip("root-owned executable fixture requires root")
			}
			executable := copyExecutable(t, helperExecutable(t))
			path := writeRegistry(t, 0o600, []testEntry{{
				ID: "root-owner", Name: "Root Owner", Executable: executable, Args: []string{},
			}})
			if _, err := loadFile(path, policyForMode(mode)); err != nil {
				t.Fatalf("root-owned executable rejected in %s mode: %v", mode, err)
			}
		})
	}
}

func TestLoadRejectsWritableExecutableInAllModes(t *testing.T) {
	tests := []struct {
		name string
		mode os.FileMode
	}{
		{name: "group writable", mode: 0o720},
		{name: "world writable", mode: 0o702},
	}
	serviceModes := []config.ServiceMode{
		config.ServiceModeSystem,
		config.ServiceModeUser,
		config.ServiceModeDevelopment,
	}

	for _, serviceMode := range serviceModes {
		for _, test := range tests {
			t.Run(string(serviceMode)+"/"+test.name, func(t *testing.T) {
				executable := copyExecutable(t, helperExecutable(t))
				if err := os.Chmod(executable, test.mode); err != nil {
					t.Fatal(err)
				}
				path := writeRegistry(t, 0o600, []testEntry{{
					ID: "writable", Name: "Writable", Executable: executable, Args: []string{},
				}})
				policy := policyForMode(serviceMode)
				if serviceMode == config.ServiceModeSystem && os.Geteuid() != 0 {
					policy.expectedOwner = uint32(os.Geteuid())
				}
				_, err := loadFile(path, policy)
				if !errors.Is(err, ErrRegistryInvalid) ||
					!strings.Contains(err.Error(), "group- or world-writable") {
					t.Fatalf("writable executable load error = %v, want mode rejection", err)
				}
			})
		}
	}
}

func TestLoadRejectsMalformedRegistryData(t *testing.T) {
	executable := helperExecutable(t)
	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed JSON", content: `{"sources":[`},
		{name: "missing sources", content: `{}`},
		{name: "null sources", content: `{"sources":null}`},
		{
			name: "duplicate ids",
			content: registryJSON(t, []testEntry{
				{ID: "duplicate", Name: "First", Executable: executable, Args: []string{}},
				{ID: "duplicate", Name: "Second", Executable: executable, Args: []string{"fixed"}},
			}),
		},
		{
			name:    "invalid id",
			content: registryJSON(t, []testEntry{{ID: "spaces are unsafe", Name: "Invalid", Executable: executable, Args: []string{}}}),
		},
		{
			name:    "overlong id",
			content: registryJSON(t, []testEntry{{ID: strings.Repeat("a", maxIDLength+1), Name: "Invalid", Executable: executable, Args: []string{}}}),
		},
		{
			name:    "missing name",
			content: registryJSON(t, []testEntry{{ID: "missing-name", Executable: executable, Args: []string{}}}),
		},
		{
			name:    "relative executable",
			content: registryJSON(t, []testEntry{{ID: "relative", Name: "Relative", Executable: "relative/program", Args: []string{}}}),
		},
		{
			name:    "nonexistent executable",
			content: registryJSON(t, []testEntry{{ID: "missing", Name: "Missing", Executable: filepath.Join(t.TempDir(), "missing"), Args: []string{}}}),
		},
		{
			name:    "missing args",
			content: fmt.Sprintf(`{"sources":[{"id":"missing-args","name":"Missing Args","executable":%q}]}`, executable),
		},
		{
			name:    "unknown field",
			content: fmt.Sprintf(`{"sources":[{"id":"unknown","name":"Unknown","executable":%q,"args":[],"environment":{}}]}`, executable),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "external-sources.json")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := loadFile(path, filePolicy{})
			if !errors.Is(err, ErrRegistryInvalid) {
				t.Fatalf("loadFile() error = %v, want ErrRegistryInvalid", err)
			}
		})
	}
}

func TestLoadRejectsUnsafeExecutableTypes(t *testing.T) {
	directory := t.TempDir()
	nonExecutable := filepath.Join(t.TempDir(), "not-executable")
	if err := os.WriteFile(nonExecutable, []byte("not executable"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		executable string
	}{
		{name: "directory", executable: directory},
		{name: "non-executable file", executable: nonExecutable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := strings.ReplaceAll(test.name, " ", "-")
			path := writeRegistry(t, 0o600, []testEntry{{
				ID: id, Name: test.name, Executable: test.executable, Args: []string{},
			}})
			_, err := loadFile(path, filePolicy{})
			if !errors.Is(err, ErrRegistryInvalid) {
				t.Fatalf("loadFile() error = %v, want ErrRegistryInvalid", err)
			}
		})
	}
}

func TestLoadRejectsRegistrySymlinkAndInsecureMode(t *testing.T) {
	target := writeRegistry(t, 0o600, []testEntry{{
		ID: "safe", Name: "Safe", Executable: helperExecutable(t), Args: []string{},
	}})
	link := filepath.Join(t.TempDir(), "external-sources.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFile(link, filePolicy{}); !errors.Is(err, ErrRegistryInvalid) {
		t.Fatalf("symlink load error = %v, want ErrRegistryInvalid", err)
	}

	insecure := writeRegistry(t, 0o622, []testEntry{{
		ID: "unsafe-mode", Name: "Unsafe", Executable: helperExecutable(t), Args: []string{},
	}})
	if _, err := loadFile(insecure, filePolicy{checkWriteMode: true}); !errors.Is(err, ErrRegistryInvalid) {
		t.Fatalf("insecure mode load error = %v, want ErrRegistryInvalid", err)
	}
}

func TestLoadRejectsIncorrectOwnership(t *testing.T) {
	path := writeRegistry(t, 0o600, []testEntry{{
		ID: "wrong-owner", Name: "Wrong Owner", Executable: helperExecutable(t), Args: []string{},
	}})
	unexpectedOwner := uint32(os.Geteuid()) + 1
	if _, err := loadFile(path, filePolicy{checkOwner: true, expectedOwner: unexpectedOwner}); !errors.Is(err, ErrRegistryInvalid) {
		t.Fatalf("incorrect owner load error = %v, want ErrRegistryInvalid", err)
	}
}

func TestLoadPreservesFixedArgumentsAndAllowsSharedExecutable(t *testing.T) {
	executable := helperExecutable(t)
	path := writeRegistry(t, 0o600, []testEntry{
		{ID: "second", Name: "Second", Executable: executable, Args: []string{"two", "ordered"}},
		{ID: "first", Name: "First", Executable: executable, Args: []string{"one", "", "three"}},
	})
	registry, err := loadFile(path, filePolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.sources["first"].args; fmt.Sprint(got) != "[one  three]" {
		t.Fatalf("first fixed args = %#v", got)
	}
	if got := registry.sources["second"].args; fmt.Sprint(got) != "[two ordered]" {
		t.Fatalf("second fixed args = %#v", got)
	}
	if got := registry.ordered; got[0].ID != "first" || got[1].ID != "second" {
		t.Fatalf("deterministic order = %#v", got)
	}
}

func TestExecuteKnownSourceUsesCanonicalPathAndFixedArgs(t *testing.T) {
	executable := helperExecutable(t)
	marker := filepath.Join(t.TempDir(), "shell-marker")
	literalArgument := "; touch " + marker
	link := filepath.Join(t.TempDir(), "helper-link")
	if err := os.Symlink(executable, link); err != nil {
		t.Fatal(err)
	}
	path := writeRegistry(t, 0o600, []testEntry{{
		ID:         "known",
		Name:       "Known",
		Executable: link,
		Args:       []string{"-test.run=^TestExternalSourceHelperProcess$", "--", "args", "fixed-one", literalArgument},
	}})
	registry, err := loadFile(path, filePolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if registry.sources["known"].executable != executable {
		t.Fatalf("canonical executable = %q, want %q", registry.sources["known"].executable, executable)
	}
	got, err := registry.Execute("known")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != 27.5 {
		t.Fatalf("Execute() = %v, want 27.5", got)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("fixed argument was interpreted by a shell; marker stat error = %v", err)
	}
}

func TestExecuteRejectsUnknownSourceWithoutExecution(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")
	registry := helperRegistry(t, "known", "marker", marker)
	if _, err := registry.Execute("unknown"); !errors.Is(err, ErrSourceUnknown) {
		t.Fatalf("Execute() error = %v, want ErrSourceUnknown", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("unknown id executed helper; marker stat error = %v", err)
	}
}

func TestExecuteTimeoutIsTwoSeconds(t *testing.T) {
	registry := helperRegistry(t, "timeout", "sleep", "10s")
	start := time.Now()
	_, err := registry.Execute("timeout")
	elapsed := time.Since(start)
	if !errors.Is(err, ErrCommandTimeout) {
		t.Fatalf("Execute() error = %v, want ErrCommandTimeout", err)
	}
	if elapsed < commandTimeout || elapsed > commandTimeout+1500*time.Millisecond {
		t.Fatalf("timeout elapsed = %v, want close to %v", elapsed, commandTimeout)
	}
}

func TestExecuteBoundsLingeringInheritedPipes(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("external source execution is supported only on Linux")
	}
	registry := helperRegistry(t, "linger", "linger-pipes", "2s")
	start := time.Now()
	_, err := registry.Execute("linger")
	elapsed := time.Since(start)
	if !errors.Is(err, ErrExecutionFailed) {
		t.Fatalf("Execute() error = %v, want ErrExecutionFailed", err)
	}
	if elapsed > commandWaitDelay+750*time.Millisecond {
		t.Fatalf("lingering inherited pipe delayed Execute() for %v", elapsed)
	}
}

func TestExecuteOutputValidation(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		value     string
		want      float32
		wantError error
	}{
		{name: "stdout too large", action: "large", wantError: ErrOutputTooLarge},
		{name: "stderr too large", action: "large-stderr", wantError: ErrOutputTooLarge},
		{name: "empty", action: "value", value: "", wantError: ErrInvalidOutput},
		{name: "malformed", action: "value", value: "not-a-number", wantError: ErrInvalidOutput},
		{name: "extra text", action: "value", value: "42 degrees", wantError: ErrInvalidOutput},
		{name: "multiple values", action: "value", value: "42\n43", wantError: ErrInvalidOutput},
		{name: "NaN", action: "value", value: "NaN", wantError: ErrInvalidOutput},
		{name: "positive infinity", action: "value", value: "+Inf", wantError: ErrInvalidOutput},
		{name: "negative infinity", action: "value", value: "-Inf", wantError: ErrInvalidOutput},
		{name: "negative", action: "value", value: "-12.25", want: -12.25},
		{name: "zero", action: "value", value: "0", want: 0},
		{name: "integer", action: "value", value: "42", want: 42},
		{name: "decimal", action: "value", value: "42.126", want: 42.13},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := helperRegistry(t, "test", test.action, test.value)
			got, err := registry.Execute("test")
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("Execute() error = %v, want %v", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Execute() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestExecuteReportsProcessFailure(t *testing.T) {
	registry := helperRegistry(t, "failure", "fail", "fixture stderr")
	if _, err := registry.Execute("failure"); !errors.Is(err, ErrExecutionFailed) {
		t.Fatalf("Execute() error = %v, want ErrExecutionFailed", err)
	}
}

func TestExecuteRevalidatesExecutable(t *testing.T) {
	tests := []struct {
		name   string
		change func(string) error
	}{
		{name: "removed", change: os.Remove},
		{name: "execute bit removed", change: func(path string) error { return os.Chmod(path, 0o600) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sourcePath := copyExecutable(t, helperExecutable(t))
			path := writeRegistry(t, 0o600, []testEntry{{
				ID:         "changed",
				Name:       "Changed",
				Executable: sourcePath,
				Args:       []string{"-test.run=^TestExternalSourceHelperProcess$", "--", "value", "20"},
			}})
			registry, err := loadFile(path, filePolicy{})
			if err != nil {
				t.Fatal(err)
			}
			if err = test.change(sourcePath); err != nil {
				t.Fatal(err)
			}
			if _, err = registry.Execute("changed"); !errors.Is(err, ErrExecutableUnavailable) {
				t.Fatalf("Execute() error = %v, want ErrExecutableUnavailable", err)
			}
		})
	}
}

func TestExecuteRejectsExecutableMadeWritableAfterLoading(t *testing.T) {
	sourcePath := copyExecutable(t, helperExecutable(t))
	path := writeRegistry(t, 0o600, []testEntry{{
		ID:         "writable-after-load",
		Name:       "Writable After Load",
		Executable: sourcePath,
		Args:       []string{"-test.run=^TestExternalSourceHelperProcess$", "--", "value", "20"},
	}})
	registry, err := loadFile(path, policyForMode(config.ServiceModeDevelopment))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(sourcePath, 0o720); err != nil {
		t.Fatal(err)
	}
	if _, err = registry.Execute("writable-after-load"); !errors.Is(err, ErrExecutableUnavailable) {
		t.Fatalf("Execute() error = %v, want ErrExecutableUnavailable", err)
	}
}

func TestExecuteRejectsExecutableOwnershipChangeAfterLoading(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing executable ownership requires root")
	}
	sourcePath := copyExecutable(t, helperExecutable(t))
	path := writeRegistry(t, 0o600, []testEntry{{
		ID:         "owner-changed",
		Name:       "Owner Changed",
		Executable: sourcePath,
		Args:       []string{"-test.run=^TestExternalSourceHelperProcess$", "--", "value", "20"},
	}})
	registry, err := loadFile(path, policyForMode(config.ServiceModeDevelopment))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chown(sourcePath, 65534, -1); err != nil {
		t.Skipf("ownership change unavailable: %v", err)
	}
	if _, err = registry.Execute("owner-changed"); !errors.Is(err, ErrExecutableUnavailable) {
		t.Fatalf("Execute() error = %v, want ErrExecutableUnavailable", err)
	}
}

func TestExternalSourceHelperProcess(t *testing.T) {
	separator := -1
	for index, arg := range os.Args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}
	args := os.Args[separator+1:]
	switch args[0] {
	case "value":
		if len(args) > 1 {
			fmt.Print(args[1])
		}
	case "args":
		if len(args) == 3 && args[1] == "fixed-one" && strings.HasPrefix(args[2], "; touch ") {
			fmt.Print("27.5")
			os.Exit(0)
		}
		os.Exit(12)
	case "marker":
		if len(args) > 1 {
			_ = os.WriteFile(args[1], []byte("executed"), 0o600)
		}
		fmt.Print("10")
	case "sleep":
		duration, _ := time.ParseDuration(args[1])
		time.Sleep(duration)
		fmt.Print("10")
	case "linger-pipes":
		command := exec.Command(
			os.Args[0],
			"-test.run=^TestExternalSourceHelperProcess$",
			"--",
			"hold-pipes",
			args[1],
		)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Start(); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(13)
		}
		fmt.Print("42.5")
	case "hold-pipes":
		duration, _ := time.ParseDuration(args[1])
		time.Sleep(duration)
	case "large":
		fmt.Print(strings.Repeat("1", maxProcessOutput+1))
	case "large-stderr":
		fmt.Fprint(os.Stderr, strings.Repeat("e", maxProcessOutput+1))
		fmt.Print("10")
	case "fail":
		if len(args) > 1 {
			fmt.Fprint(os.Stderr, args[1])
		}
		os.Exit(9)
	}
	os.Exit(0)
}

type testEntry struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

func helperExecutable(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func helperRegistry(t *testing.T, id, action string, values ...string) Registry {
	t.Helper()
	args := []string{"-test.run=^TestExternalSourceHelperProcess$", "--", action}
	args = append(args, values...)
	path := writeRegistry(t, 0o600, []testEntry{{
		ID: id, Name: id, Executable: helperExecutable(t), Args: args,
	}})
	registry, err := loadFile(path, filePolicy{})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func writeRegistry(t *testing.T, mode os.FileMode, entries []testEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "external-sources.json")
	content := registryJSON(t, entries)
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func registryJSON(t *testing.T, entries []testEntry) string {
	t.Helper()
	data, err := json.Marshal(struct {
		Sources []testEntry `json:"sources"`
	}{Sources: entries})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func copyExecutable(t *testing.T, source string) string {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "helper-copy")
	if err = os.WriteFile(target, data, 0o700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "linux" {
		t.Skip("external source execution is supported only on Linux")
	}
	return target
}
