package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	ServiceModeDevelopment ServiceMode = "development"
	ServiceModeSystem      ServiceMode = "system"
	ServiceModeUser        ServiceMode = "user"

	defaultApplicationRoot = "/opt/LumenForge"
	defaultSystemStateRoot = "/var/lib/lumenforge"
)

// ServiceMode identifies how LumenForge was launched. Installed modes never
// derive paths from the process working directory.
type ServiceMode string

// PathOptions contains the inputs used to resolve the complete filesystem
// layout. Tests can provide explicit temporary roots without changing global
// environment variables or touching installed paths.
type PathOptions struct {
	Mode             ServiceMode
	ApplicationRoot  string
	ConfigRoot       string
	DataRoot         string
	HomeDirectory    string
	WorkingDirectory string
	XDGConfigHome    string
	XDGDataHome      string
}

// Paths is the single source of truth for immutable application resources and
// mutable runtime state.
type Paths struct {
	Mode ServiceMode

	ApplicationRoot           string
	ConfigurationDirectory    string
	ConfigurationFile         string
	ExternalSourcesFile       string
	MutableDataRoot           string
	MutableDatabaseRoot       string
	DashboardFile             string
	DisplayFile               string
	MutableProfilesRoot       string
	MutableRGBRoot            string
	MutableTemperaturesRoot   string
	MutableMacrosRoot         string
	MutableKeyAssignmentsRoot string
	MutableLEDRoot            string
	OpenRGBImportFile         string

	ShippedDatabaseRoot          string
	ShippedDeviceDefinitionsRoot string
	FrontendRoot                 string
	TemplateRoot                 string
	StaticAssetRoot              string
	ShippedLCDRoot               string
	ShippedLCDMediaRoot          string
	MutableLCDRoot               string
	MutableLCDUploadRoot         string

	BackupConfigurationFile  string
	BackupDataRoot           string
	RestoreConfigurationRoot string
	RestoreDataRoot          string
	DefaultLogDestination    string
}

var runtimePaths Paths

// ResolveRuntimePaths resolves paths from the process environment. An unset
// service mode is an intentional direct-development run.
func ResolveRuntimePaths() (Paths, error) {
	modeValue := strings.ToLower(strings.TrimSpace(os.Getenv("LUMENFORGE_SERVICE_MODE")))
	mode := ServiceModeDevelopment
	switch modeValue {
	case "":
	case string(ServiceModeSystem):
		mode = ServiceModeSystem
	case string(ServiceModeUser), "desktop":
		mode = ServiceModeUser
	default:
		return Paths{}, fmt.Errorf("unsupported LUMENFORGE_SERVICE_MODE %q", modeValue)
	}

	workingDirectory := ""
	if mode == ServiceModeDevelopment {
		var err error
		workingDirectory, err = os.Getwd()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve development working directory: %w", err)
		}
	}

	homeDirectory := ""
	if mode == ServiceModeUser {
		currentUser, userErr := user.Current()
		if userErr != nil {
			return Paths{}, fmt.Errorf("resolve current user: %w", userErr)
		}
		homeDirectory = currentUser.HomeDir
	}

	return ResolvePaths(PathOptions{
		Mode:             mode,
		ApplicationRoot:  os.Getenv("LUMENFORGE_APPLICATION_ROOT"),
		ConfigRoot:       os.Getenv("LUMENFORGE_CONFIG_ROOT"),
		DataRoot:         os.Getenv("LUMENFORGE_DATA_ROOT"),
		HomeDirectory:    homeDirectory,
		WorkingDirectory: workingDirectory,
		XDGConfigHome:    os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:      os.Getenv("XDG_DATA_HOME"),
	})
}

// ResolvePaths validates and constructs the complete path model.
func ResolvePaths(options PathOptions) (Paths, error) {
	mode := options.Mode
	if mode == "" {
		mode = ServiceModeDevelopment
	}
	if mode != ServiceModeDevelopment && mode != ServiceModeSystem && mode != ServiceModeUser {
		return Paths{}, fmt.Errorf("unsupported service mode %q", mode)
	}

	var applicationRoot, configRoot, dataRoot string
	switch mode {
	case ServiceModeDevelopment:
		workingDirectory := options.WorkingDirectory
		if workingDirectory == "" {
			return Paths{}, fmt.Errorf("development mode requires a working directory")
		}
		applicationRoot = firstNonEmpty(options.ApplicationRoot, workingDirectory)
		configRoot = firstNonEmpty(options.ConfigRoot, workingDirectory)
		dataRoot = firstNonEmpty(options.DataRoot, workingDirectory)
	case ServiceModeSystem:
		applicationRoot = firstNonEmpty(options.ApplicationRoot, defaultApplicationRoot)
		configRoot = firstNonEmpty(options.ConfigRoot, defaultSystemStateRoot)
		dataRoot = firstNonEmpty(options.DataRoot, defaultSystemStateRoot)
	case ServiceModeUser:
		applicationRoot = firstNonEmpty(options.ApplicationRoot, defaultApplicationRoot)
		configRoot = options.ConfigRoot
		if configRoot == "" {
			configHome := options.XDGConfigHome
			if configHome == "" {
				if options.HomeDirectory == "" {
					return Paths{}, fmt.Errorf("user mode requires a home directory when XDG_CONFIG_HOME is unset")
				}
				configHome = filepath.Join(options.HomeDirectory, ".config")
			}
			configRoot = filepath.Join(configHome, "lumenforge")
		}
		dataRoot = options.DataRoot
		if dataRoot == "" {
			dataHome := options.XDGDataHome
			if dataHome == "" {
				if options.HomeDirectory == "" {
					return Paths{}, fmt.Errorf("user mode requires a home directory when XDG_DATA_HOME is unset")
				}
				dataHome = filepath.Join(options.HomeDirectory, ".local", "share")
			}
			dataRoot = filepath.Join(dataHome, "lumenforge")
		}
	}

	var err error
	if applicationRoot, err = cleanAbsolutePath("application root", applicationRoot); err != nil {
		return Paths{}, err
	}
	if configRoot, err = cleanAbsolutePath("configuration root", configRoot); err != nil {
		return Paths{}, err
	}
	if dataRoot, err = cleanAbsolutePath("mutable data root", dataRoot); err != nil {
		return Paths{}, err
	}

	if mode != ServiceModeDevelopment {
		if pathWithin(configRoot, applicationRoot) {
			return Paths{}, fmt.Errorf("configuration root %q must not be inside application root %q", configRoot, applicationRoot)
		}
		if pathWithin(dataRoot, applicationRoot) {
			return Paths{}, fmt.Errorf("mutable data root %q must not be inside application root %q", dataRoot, applicationRoot)
		}
	}

	shippedDatabaseRoot := filepath.Join(applicationRoot, "database")
	mutableDatabaseRoot := filepath.Join(dataRoot, "database")
	shippedLCDRoot := filepath.Join(shippedDatabaseRoot, "lcd")
	mutableLCDRoot := filepath.Join(mutableDatabaseRoot, "lcd")
	externalSourcesFile := filepath.Join(configRoot, "external-sources.json")
	if mode == ServiceModeSystem {
		externalSourcesFile = "/etc/lumenforge/external-sources.json"
	}

	return Paths{
		Mode: mode,

		ApplicationRoot:           applicationRoot,
		ConfigurationDirectory:    configRoot,
		ConfigurationFile:         filepath.Join(configRoot, "config.json"),
		ExternalSourcesFile:       externalSourcesFile,
		MutableDataRoot:           dataRoot,
		MutableDatabaseRoot:       mutableDatabaseRoot,
		DashboardFile:             filepath.Join(dataRoot, "dashboard.json"),
		DisplayFile:               filepath.Join(dataRoot, "display.json"),
		MutableProfilesRoot:       filepath.Join(mutableDatabaseRoot, "profiles"),
		MutableRGBRoot:            filepath.Join(mutableDatabaseRoot, "rgb"),
		MutableTemperaturesRoot:   filepath.Join(mutableDatabaseRoot, "temperatures"),
		MutableMacrosRoot:         filepath.Join(mutableDatabaseRoot, "macros"),
		MutableKeyAssignmentsRoot: filepath.Join(mutableDatabaseRoot, "key-assignments"),
		MutableLEDRoot:            filepath.Join(mutableDatabaseRoot, "led"),
		OpenRGBImportFile:         filepath.Join(mutableDatabaseRoot, "openrgbimport-zones.json"),

		ShippedDatabaseRoot:          shippedDatabaseRoot,
		ShippedDeviceDefinitionsRoot: filepath.Join(shippedDatabaseRoot, "external"),
		FrontendRoot:                 applicationRoot,
		TemplateRoot:                 filepath.Join(applicationRoot, "web"),
		StaticAssetRoot:              filepath.Join(applicationRoot, "static"),
		ShippedLCDRoot:               shippedLCDRoot,
		ShippedLCDMediaRoot:          filepath.Join(shippedLCDRoot, "images"),
		MutableLCDRoot:               mutableLCDRoot,
		MutableLCDUploadRoot:         filepath.Join(mutableLCDRoot, "images"),

		BackupConfigurationFile:  filepath.Join(configRoot, "config.json"),
		BackupDataRoot:           dataRoot,
		RestoreConfigurationRoot: configRoot,
		RestoreDataRoot:          dataRoot,
		DefaultLogDestination:    "-",
	}, nil
}

// EnsureRuntimeDirectories creates only mutable directories.
func EnsureRuntimeDirectories(paths Paths) error {
	baseMode := os.FileMode(0o700)
	if paths.Mode == ServiceModeSystem {
		baseMode = 0o750
	}
	if err := os.MkdirAll(paths.ConfigurationDirectory, baseMode); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	if err := os.MkdirAll(paths.MutableDataRoot, baseMode); err != nil {
		return fmt.Errorf("create mutable data directory: %w", err)
	}

	for _, directory := range []string{
		paths.MutableDatabaseRoot,
		paths.MutableKeyAssignmentsRoot,
		paths.MutableLEDRoot,
		paths.MutableMacrosRoot,
		paths.MutableProfilesRoot,
		paths.MutableRGBRoot,
		paths.MutableTemperaturesRoot,
		paths.MutableLCDRoot,
		paths.MutableLCDUploadRoot,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create mutable runtime directory %q: %w", directory, err)
		}
	}
	return nil
}

// ResolveLogFile maps the configured logging value to a safe destination.
func (paths Paths) ResolveLogFile(configured string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return paths.DefaultLogDestination, nil
	}
	if configured == "-" {
		return configured, nil
	}

	resolved := configured
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(paths.MutableDataRoot, resolved)
	}
	var err error
	if resolved, err = cleanAbsolutePath("log file", resolved); err != nil {
		return "", err
	}
	if pathWithin(resolved, paths.ApplicationRoot) {
		return "", fmt.Errorf("log file %q must not be inside application root %q", resolved, paths.ApplicationRoot)
	}
	return resolved, nil
}

// GetPaths returns the initialized process path model.
func GetPaths() Paths {
	return runtimePaths
}

// UsePathsForTest installs a temporary path model and returns a restore
// function. Tests using it must not run in parallel.
func UsePathsForTest(paths Paths) func() {
	previous := runtimePaths
	runtimePaths = paths
	return func() {
		runtimePaths = previous
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func cleanAbsolutePath(name, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("%s %q must be absolute", name, value)
	}

	cleaned := filepath.Clean(value)
	probe := cleaned
	var missingComponents []string
	for {
		_, err := os.Lstat(probe)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("resolve %s %q: %w", name, cleaned, err)
		}

		parent := filepath.Dir(probe)
		if parent == probe {
			return "", fmt.Errorf("resolve %s %q: no existing path ancestor", name, cleaned)
		}
		missingComponents = append(missingComponents, filepath.Base(probe))
		probe = parent
	}

	canonical, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", name, cleaned, err)
	}
	for index := len(missingComponents) - 1; index >= 0; index-- {
		canonical = filepath.Join(canonical, missingComponents[index])
	}
	return filepath.Clean(canonical), nil
}

func pathWithin(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
}
