package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolvePathsInstalledUserWithXDGRoots(t *testing.T) {
	paths, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeUser,
		HomeDirectory:   "/home/alice",
		XDGConfigHome:   "/srv/alice-config",
		XDGDataHome:     "/srv/alice-data",
		ApplicationRoot: "/opt/LumenForge",
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.ConfigurationFile != "/srv/alice-config/lumenforge/config.json" {
		t.Fatalf("ConfigurationFile = %q", paths.ConfigurationFile)
	}
	if paths.ExternalSourcesFile != "/srv/alice-config/lumenforge/external-sources.json" {
		t.Fatalf("ExternalSourcesFile = %q", paths.ExternalSourcesFile)
	}
	if paths.MutableDataRoot != "/srv/alice-data/lumenforge" {
		t.Fatalf("MutableDataRoot = %q", paths.MutableDataRoot)
	}
	if paths.MutableDatabaseRoot != "/srv/alice-data/lumenforge/database" {
		t.Fatalf("MutableDatabaseRoot = %q", paths.MutableDatabaseRoot)
	}
}

func TestResolvePathsInstalledUserFallbacks(t *testing.T) {
	paths, err := ResolvePaths(PathOptions{
		Mode:          ServiceModeUser,
		HomeDirectory: "/home/alice",
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	if paths.ConfigurationFile != "/home/alice/.config/lumenforge/config.json" {
		t.Fatalf("ConfigurationFile = %q", paths.ConfigurationFile)
	}
	if paths.MutableDataRoot != "/home/alice/.local/share/lumenforge" {
		t.Fatalf("MutableDataRoot = %q", paths.MutableDataRoot)
	}
}

func TestResolvePathsInstalledSystem(t *testing.T) {
	paths, err := ResolvePaths(PathOptions{Mode: ServiceModeSystem})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	checks := map[string]string{
		"application": paths.ApplicationRoot,
		"config":      paths.ConfigurationFile,
		"external":    paths.ExternalSourcesFile,
		"data":        paths.MutableDataRoot,
		"database":    paths.MutableDatabaseRoot,
		"templates":   paths.TemplateRoot,
		"static":      paths.StaticAssetRoot,
	}
	wants := map[string]string{
		"application": "/opt/LumenForge",
		"config":      "/var/lib/lumenforge/config.json",
		"external":    "/etc/lumenforge/external-sources.json",
		"data":        "/var/lib/lumenforge",
		"database":    "/var/lib/lumenforge/database",
		"templates":   "/opt/LumenForge/web",
		"static":      "/opt/LumenForge/static",
	}
	for name, got := range checks {
		if got != wants[name] {
			t.Errorf("%s path = %q, want %q", name, got, wants[name])
		}
	}
}

func TestResolvePathsDevelopmentAndExplicitTemporaryRoots(t *testing.T) {
	workingDirectory := t.TempDir()
	paths, err := ResolvePaths(PathOptions{
		Mode:             ServiceModeDevelopment,
		WorkingDirectory: workingDirectory,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	if paths.ApplicationRoot != workingDirectory || paths.ConfigurationDirectory != workingDirectory || paths.MutableDataRoot != workingDirectory {
		t.Fatalf("development roots = %#v", paths)
	}
	if paths.ExternalSourcesFile != filepath.Join(workingDirectory, "external-sources.json") {
		t.Fatalf("development ExternalSourcesFile = %q", paths.ExternalSourcesFile)
	}

	root := t.TempDir()
	paths, err = ResolvePaths(PathOptions{
		Mode:            ServiceModeUser,
		ApplicationRoot: filepath.Join(root, "app"),
		ConfigRoot:      filepath.Join(root, "config", "..", "config"),
		DataRoot:        filepath.Join(root, "data", ".", "state"),
	})
	if err != nil {
		t.Fatalf("ResolvePaths(explicit roots) error = %v", err)
	}
	if paths.ConfigurationDirectory != filepath.Join(root, "config") {
		t.Fatalf("cleaned configuration root = %q", paths.ConfigurationDirectory)
	}
	if paths.MutableDataRoot != filepath.Join(root, "data", "state") {
		t.Fatalf("cleaned data root = %q", paths.MutableDataRoot)
	}
}

func TestResolvePathsRejectsInvalidInstalledRoots(t *testing.T) {
	tests := []struct {
		name    string
		options PathOptions
		want    string
	}{
		{
			name: "relative XDG config",
			options: PathOptions{
				Mode:          ServiceModeUser,
				HomeDirectory: "/home/alice",
				XDGConfigHome: "relative-config",
				XDGDataHome:   "/data/alice",
			},
			want: "configuration root",
		},
		{
			name: "relative XDG data",
			options: PathOptions{
				Mode:          ServiceModeUser,
				HomeDirectory: "/home/alice",
				XDGConfigHome: "/config/alice",
				XDGDataHome:   "relative-data",
			},
			want: "mutable data root",
		},
		{
			name: "config inside application",
			options: PathOptions{
				Mode:            ServiceModeSystem,
				ApplicationRoot: "/opt/LumenForge",
				ConfigRoot:      "/opt/LumenForge/config",
				DataRoot:        "/var/lib/lumenforge",
			},
			want: "must not be inside application root",
		},
		{
			name: "data inside application",
			options: PathOptions{
				Mode:            ServiceModeSystem,
				ApplicationRoot: "/opt/LumenForge",
				ConfigRoot:      "/var/lib/lumenforge",
				DataRoot:        "/opt/LumenForge/database/runtime",
			},
			want: "must not be inside application root",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolvePaths(test.options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ResolvePaths() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestResolvePathsRejectsSymlinkedInstalledRootInsideApplication(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	if err := os.MkdirAll(filepath.Join(applicationRoot, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	configLink := filepath.Join(root, "config-link")
	if err := os.Symlink(filepath.Join(applicationRoot, "runtime"), configLink); err != nil {
		t.Fatal(err)
	}

	_, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeSystem,
		ApplicationRoot: applicationRoot,
		ConfigRoot:      configLink,
		DataRoot:        filepath.Join(root, "data"),
	})
	if err == nil || !strings.Contains(err.Error(), "must not be inside application root") {
		t.Fatalf("ResolvePaths() error = %v, want symlink escape rejection", err)
	}
}

func TestInstalledResolutionDoesNotDependOnWorkingDirectory(t *testing.T) {
	first, err := ResolvePaths(PathOptions{
		Mode:             ServiceModeSystem,
		WorkingDirectory: "/tmp/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolvePaths(PathOptions{
		Mode:             ServiceModeSystem,
		WorkingDirectory: "/tmp/second",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("system paths changed with working directory:\nfirst: %#v\nsecond: %#v", first, second)
	}

	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWorkingDirectory)
	})
	t.Setenv("LUMENFORGE_SERVICE_MODE", "system")
	t.Setenv("LUMENFORGE_APPLICATION_ROOT", filepath.Join(t.TempDir(), "app"))
	t.Setenv("LUMENFORGE_CONFIG_ROOT", filepath.Join(t.TempDir(), "config"))
	t.Setenv("LUMENFORGE_DATA_ROOT", filepath.Join(t.TempDir(), "data"))

	firstWorkingDirectory := t.TempDir()
	secondWorkingDirectory := t.TempDir()
	if err = os.Chdir(firstWorkingDirectory); err != nil {
		t.Fatal(err)
	}
	firstRuntime, err := ResolveRuntimePaths()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(secondWorkingDirectory); err != nil {
		t.Fatal(err)
	}
	secondRuntime, err := ResolveRuntimePaths()
	if err != nil {
		t.Fatal(err)
	}
	if firstRuntime != secondRuntime {
		t.Fatalf("runtime system paths changed with working directory:\nfirst: %#v\nsecond: %#v", firstRuntime, secondRuntime)
	}
}

func TestResolveRuntimePathsModeSelection(t *testing.T) {
	workingDirectory := t.TempDir()
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWorkingDirectory)
	})

	t.Setenv("LUMENFORGE_SERVICE_MODE", "")
	t.Setenv("LUMENFORGE_APPLICATION_ROOT", "")
	t.Setenv("LUMENFORGE_CONFIG_ROOT", "")
	t.Setenv("LUMENFORGE_DATA_ROOT", "")
	paths, err := ResolveRuntimePaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.Mode != ServiceModeDevelopment || paths.ApplicationRoot != workingDirectory {
		t.Fatalf("direct paths = %#v", paths)
	}

	t.Setenv("LUMENFORGE_SERVICE_MODE", "unsupported")
	if _, err = ResolveRuntimePaths(); err == nil {
		t.Fatal("ResolveRuntimePaths() accepted unsupported service mode")
	}
}

func TestResolveLogFile(t *testing.T) {
	paths, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeUser,
		ApplicationRoot: "/opt/LumenForge",
		ConfigRoot:      "/tmp/config",
		DataRoot:        "/tmp/data",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, err := paths.ResolveLogFile(""); err != nil || got != "-" {
		t.Fatalf("empty log destination = %q, %v", got, err)
	}
	if got, err := paths.ResolveLogFile("logs/lumenforge.log"); err != nil || got != "/tmp/data/logs/lumenforge.log" {
		t.Fatalf("relative log destination = %q, %v", got, err)
	}
	if _, err := paths.ResolveLogFile("/opt/LumenForge/runtime.log"); err == nil {
		t.Fatal("ResolveLogFile() accepted application-tree destination")
	}
}

func TestEnsureRuntimeDirectoriesKeepsMutableCategoriesOutsideApplication(t *testing.T) {
	root := t.TempDir()
	applicationRoot := filepath.Join(root, "app")
	if err := os.Mkdir(applicationRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(applicationRoot, 0o755)
	})

	paths, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeUser,
		ApplicationRoot: applicationRoot,
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(applicationRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	if err = EnsureRuntimeDirectories(paths); err != nil {
		t.Fatal(err)
	}

	mutableDirectories := []string{
		paths.MutableProfilesRoot,
		paths.MutableRGBRoot,
		paths.MutableTemperaturesRoot,
		paths.MutableMacrosRoot,
		paths.MutableKeyAssignmentsRoot,
		paths.MutableLEDRoot,
		paths.MutableLCDRoot,
		paths.MutableLCDUploadRoot,
	}
	for _, directory := range mutableDirectories {
		if !pathWithin(directory, paths.MutableDataRoot) {
			t.Errorf("mutable directory %q is outside data root %q", directory, paths.MutableDataRoot)
			continue
		}
		if pathWithin(directory, paths.ApplicationRoot) {
			t.Errorf("mutable directory %q is inside application root", directory)
			continue
		}
		if err = os.WriteFile(filepath.Join(directory, "write-test.json"), []byte("{}"), 0o600); err != nil {
			t.Errorf("write representative mutable file in %q: %v", directory, err)
		}
	}
	if err = os.WriteFile(paths.OpenRGBImportFile, []byte("{}"), 0o600); err != nil {
		t.Errorf("write OpenRGB import file: %v", err)
	}
	if entries, readErr := os.ReadDir(paths.ApplicationRoot); readErr != nil {
		t.Fatal(readErr)
	} else if len(entries) != 0 {
		t.Fatalf("runtime directory creation wrote into application root: %v", entries)
	}
}

func TestInitWithPathsCreatesConfigAtResolvedLocation(t *testing.T) {
	root := t.TempDir()
	paths, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeUser,
		ApplicationRoot: filepath.Join(root, "app"),
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}

	originalPaths := runtimePaths
	originalLocation := location
	originalConfiguration := configuration
	originalSystemService := systemService
	t.Cleanup(func() {
		runtimePaths = originalPaths
		location = originalLocation
		configuration = originalConfiguration
		systemService = originalSystemService
	})

	initWithPaths(paths)

	info, err := os.Stat(paths.ConfigurationFile)
	if err != nil {
		t.Fatalf("config was not created at %q: %v", paths.ConfigurationFile, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config mode = %o, want no group/world permissions", info.Mode().Perm())
	}
	data, err := os.ReadFile(paths.ConfigurationFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ApplicationRoot", "ConfigRoot", "DataRoot", "ConfigPath"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("config.json contains internal path field %q", forbidden)
		}
	}

	UpdateSupportedDevices(map[uint16]bool{1234: false})
	data, err = os.ReadFile(paths.ConfigurationFile)
	if err != nil {
		t.Fatal(err)
	}
	var persisted Configuration
	if err = json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(persisted.Exclude, 1234) {
		t.Fatalf("supported-device exclusion was not persisted at %q", paths.ConfigurationFile)
	}
}

func TestInitWithPathsUpgradesConfigAtResolvedLocation(t *testing.T) {
	root := t.TempDir()
	paths, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeUser,
		ApplicationRoot: filepath.Join(root, "app"),
		ConfigRoot:      filepath.Join(root, "config"),
		DataRoot:        filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(paths.ConfigurationDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.ConfigurationFile, []byte(`{"listenPort":28080}`), 0o644); err != nil {
		t.Fatal(err)
	}

	originalPaths := runtimePaths
	originalLocation := location
	originalConfiguration := configuration
	originalSystemService := systemService
	t.Cleanup(func() {
		runtimePaths = originalPaths
		location = originalLocation
		configuration = originalConfiguration
		systemService = originalSystemService
	})

	initWithPaths(paths)

	data, err := os.ReadFile(paths.ConfigurationFile)
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]any
	if err = json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted["resumeDelay"] != float64(15000) {
		t.Fatalf("upgraded resumeDelay = %#v, want 15000", persisted["resumeDelay"])
	}
	if persisted["listenPort"] != float64(28080) {
		t.Fatalf("existing listenPort = %#v, want 28080", persisted["listenPort"])
	}
	info, err := os.Stat(paths.ConfigurationFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("upgraded config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestInitWithPathsCreatesSystemConfigAtStateRoot(t *testing.T) {
	root := t.TempDir()
	paths, err := ResolvePaths(PathOptions{
		Mode:            ServiceModeSystem,
		ApplicationRoot: filepath.Join(root, "app"),
		ConfigRoot:      filepath.Join(root, "state"),
		DataRoot:        filepath.Join(root, "state"),
	})
	if err != nil {
		t.Fatal(err)
	}

	originalPaths := runtimePaths
	originalLocation := location
	originalConfiguration := configuration
	originalSystemService := systemService
	t.Cleanup(func() {
		runtimePaths = originalPaths
		location = originalLocation
		configuration = originalConfiguration
		systemService = originalSystemService
	})

	initWithPaths(paths)
	if location != filepath.Join(root, "state", "config.json") {
		t.Fatalf("system config location = %q", location)
	}
	if !IsSystemService() {
		t.Fatal("system paths did not select system service behavior")
	}
}
