package externalsources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"LumenForge/src/config"
)

const (
	maxIDLength      = 64
	maxNameLength    = 128
	maxProcessOutput = 4 * 1024
	commandTimeout   = 2 * time.Second
	commandWaitDelay = 250 * time.Millisecond
)

var (
	ErrRegistryMissing       = errors.New("external source registry is missing")
	ErrRegistryInvalid       = errors.New("external source registry is invalid")
	ErrSourceUnknown         = errors.New("external source id is unknown")
	ErrExecutableUnavailable = errors.New("external source executable is unavailable")
	ErrCommandTimeout        = errors.New("external source command timed out")
	ErrOutputTooLarge        = errors.New("external source output is too large")
	ErrExecutionFailed       = errors.New("external source process execution failed")
	ErrInvalidOutput         = errors.New("external source output is not a finite numeric value")

	idPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// Info is the browser-safe representation of a registered external source.
type Info struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type registryFile struct {
	Sources *[]registryEntry `json:"sources"`
}

type registryEntry struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Executable string    `json:"executable"`
	Args       *[]string `json:"args"`
}

type source struct {
	info       Info
	executable string
	args       []string
	fileInfo   os.FileInfo
	policy     executablePolicy
}

// Registry is an immutable validated snapshot of external-sources.json.
type Registry struct {
	sources map[string]source
	ordered []Info
	missing bool
}

type filePolicy struct {
	checkOwner     bool
	expectedOwner  uint32
	checkWriteMode bool
	executable     executablePolicy
}

type executablePolicy struct {
	allowedOwners  []uint32
	checkOwner     bool
	checkWriteMode bool
}

// Load reads and validates the single registry selected by the configured
// service mode. A missing file is a valid empty registry.
func Load(paths config.Paths) (Registry, error) {
	return loadFile(paths.ExternalSourcesFile, policyForMode(paths.Mode))
}

// List returns only information safe for the browser. The bool reports whether
// the optional registry file is absent.
func List(paths config.Paths) ([]Info, bool, error) {
	registry, err := Load(paths)
	if err != nil {
		return nil, false, err
	}
	entries := make([]Info, len(registry.ordered))
	copy(entries, registry.ordered)
	return entries, registry.missing, nil
}

// ValidateSelection verifies that an id resolves through the trusted registry.
func ValidateSelection(paths config.Paths, id string) error {
	registry, err := Load(paths)
	if err != nil {
		return err
	}
	_, err = registry.lookup(id)
	return err
}

// Execute loads the registry on demand and executes the selected source.
func Execute(paths config.Paths, id string) (float32, error) {
	registry, err := Load(paths)
	if err != nil {
		return 0, err
	}
	return registry.Execute(id)
}

// Execute runs a source from this validated snapshot. It revalidates the
// canonical executable immediately before starting it.
func (registry Registry) Execute(id string) (float32, error) {
	selected, err := registry.lookup(id)
	if err != nil {
		return 0, err
	}
	if err = revalidateExecutable(selected); err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	stdout := &limitedCapture{limit: maxProcessOutput}
	stderr := &limitedCapture{limit: maxProcessOutput}
	command := exec.CommandContext(ctx, selected.executable, selected.args...)
	// Bound cleanup when a descendant inherits and retains stdout or stderr.
	command.WaitDelay = commandWaitDelay
	command.Stdout = stdout
	command.Stderr = stderr
	runErr := command.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return 0, fmt.Errorf("%w: source %q", ErrCommandTimeout, selected.info.ID)
	}
	if stdout.tooLarge || stderr.tooLarge {
		return 0, fmt.Errorf("%w: source %q", ErrOutputTooLarge, selected.info.ID)
	}
	if runErr != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return 0, fmt.Errorf("%w: source %q: %v: %s", ErrExecutionFailed, selected.info.ID, runErr, detail)
		}
		return 0, fmt.Errorf("%w: source %q: %v", ErrExecutionFailed, selected.info.ID, runErr)
	}

	return parseTemperature(stdout.Bytes())
}

func policyForMode(mode config.ServiceMode) filePolicy {
	currentOwner := uint32(os.Geteuid())
	switch mode {
	case config.ServiceModeSystem:
		return filePolicy{
			checkOwner:     true,
			expectedOwner:  0,
			checkWriteMode: true,
			executable: executablePolicy{
				allowedOwners:  []uint32{0},
				checkOwner:     true,
				checkWriteMode: true,
			},
		}
	case config.ServiceModeUser:
		return filePolicy{
			checkOwner:     true,
			expectedOwner:  currentOwner,
			checkWriteMode: true,
			executable: executablePolicy{
				allowedOwners:  []uint32{currentOwner, 0},
				checkOwner:     true,
				checkWriteMode: true,
			},
		}
	case config.ServiceModeDevelopment:
		return filePolicy{
			executable: executablePolicy{
				allowedOwners:  []uint32{currentOwner, 0},
				checkOwner:     true,
				checkWriteMode: true,
			},
		}
	default:
		return filePolicy{}
	}
}

func loadFile(path string, policy filePolicy) (Registry, error) {
	empty := Registry{sources: make(map[string]source)}
	fileInfo, err := os.Lstat(path)
	if os.IsNotExist(err) {
		empty.missing = true
		return empty, nil
	}
	if err != nil {
		return Registry{}, fmt.Errorf("%w: inspect registry: %v", ErrRegistryInvalid, err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return Registry{}, fmt.Errorf("%w: registry must not be a symlink", ErrRegistryInvalid)
	}
	if !fileInfo.Mode().IsRegular() {
		return Registry{}, fmt.Errorf("%w: registry must be a regular file", ErrRegistryInvalid)
	}
	if policy.checkWriteMode && fileInfo.Mode().Perm()&0o022 != 0 {
		return Registry{}, fmt.Errorf("%w: registry must not be group- or world-writable", ErrRegistryInvalid)
	}
	if policy.checkOwner {
		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return Registry{}, fmt.Errorf("%w: registry ownership is unavailable", ErrRegistryInvalid)
		}
		if stat.Uid != policy.expectedOwner {
			return Registry{}, fmt.Errorf(
				"%w: registry owner is %d, expected %d",
				ErrRegistryInvalid,
				stat.Uid,
				policy.expectedOwner,
			)
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return Registry{}, fmt.Errorf("%w: open registry: %v", ErrRegistryInvalid, err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return Registry{}, fmt.Errorf("%w: inspect opened registry: %v", ErrRegistryInvalid, err)
	}
	if !os.SameFile(fileInfo, openedInfo) {
		return Registry{}, fmt.Errorf("%w: registry changed while opening", ErrRegistryInvalid)
	}

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var decoded registryFile
	if err = decoder.Decode(&decoded); err != nil {
		return Registry{}, fmt.Errorf("%w: decode registry: %v", ErrRegistryInvalid, err)
	}
	if err = requireJSONEOF(decoder); err != nil {
		return Registry{}, fmt.Errorf("%w: decode registry: %v", ErrRegistryInvalid, err)
	}
	if decoded.Sources == nil {
		return Registry{}, fmt.Errorf("%w: sources must be a JSON array", ErrRegistryInvalid)
	}

	registry := Registry{
		sources: make(map[string]source, len(*decoded.Sources)),
		ordered: make([]Info, 0, len(*decoded.Sources)),
	}
	for index, entry := range *decoded.Sources {
		validated, validateErr := validateEntry(entry, policy.executable)
		if validateErr != nil {
			return Registry{}, fmt.Errorf("%w: source %d: %v", ErrRegistryInvalid, index, validateErr)
		}
		if _, exists := registry.sources[validated.info.ID]; exists {
			return Registry{}, fmt.Errorf("%w: duplicate source id %q", ErrRegistryInvalid, validated.info.ID)
		}
		registry.sources[validated.info.ID] = validated
		registry.ordered = append(registry.ordered, validated.info)
	}
	sort.Slice(registry.ordered, func(i, j int) bool {
		return registry.ordered[i].ID < registry.ordered[j].ID
	})
	return registry, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra json.RawMessage
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("unexpected data after registry object")
	}
	return err
}

func validateEntry(entry registryEntry, policy executablePolicy) (source, error) {
	if entry.ID == "" || len(entry.ID) > maxIDLength || !idPattern.MatchString(entry.ID) {
		return source{}, fmt.Errorf("id must contain 1-%d letters, numbers, dots, underscores, or hyphens", maxIDLength)
	}
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		return source{}, errors.New("name must not be empty")
	}
	if len(name) > maxNameLength {
		return source{}, fmt.Errorf("name must not exceed %d bytes", maxNameLength)
	}
	if entry.Args == nil {
		return source{}, errors.New("args must be a JSON string array")
	}
	executable, fileInfo, err := validateExecutable(entry.Executable, policy)
	if err != nil {
		return source{}, err
	}
	return source{
		info:       Info{ID: entry.ID, Name: name},
		executable: executable,
		args:       append([]string(nil), (*entry.Args)...),
		fileInfo:   fileInfo,
		policy:     policy,
	}, nil
}

func validateExecutable(path string, policy executablePolicy) (string, os.FileInfo, error) {
	if !filepath.IsAbs(path) {
		return "", nil, errors.New("executable must be an absolute path")
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", nil, fmt.Errorf("resolve executable: %w", err)
	}
	fileInfo, err := os.Stat(canonical)
	if err != nil {
		return "", nil, fmt.Errorf("inspect executable: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return "", nil, errors.New("executable must be a regular file")
	}
	if fileInfo.Mode().Perm()&0o111 == 0 {
		return "", nil, errors.New("executable file has no execute bit")
	}
	if err = validateExecutableTrust(fileInfo, policy); err != nil {
		return "", nil, err
	}
	return canonical, fileInfo, nil
}

func revalidateExecutable(selected source) error {
	canonical, fileInfo, err := validateExecutable(selected.executable, selected.policy)
	if err != nil {
		return fmt.Errorf("%w: source %q: %v", ErrExecutableUnavailable, selected.info.ID, err)
	}
	if canonical != selected.executable ||
		!os.SameFile(fileInfo, selected.fileInfo) ||
		!sameOwnership(fileInfo, selected.fileInfo) ||
		fileInfo.Mode() != selected.fileInfo.Mode() ||
		fileInfo.Size() != selected.fileInfo.Size() ||
		!fileInfo.ModTime().Equal(selected.fileInfo.ModTime()) {
		return fmt.Errorf("%w: source %q changed after registry validation", ErrExecutableUnavailable, selected.info.ID)
	}
	return nil
}

func validateExecutableTrust(fileInfo os.FileInfo, policy executablePolicy) error {
	if policy.checkWriteMode && fileInfo.Mode().Perm()&0o022 != 0 {
		return errors.New("executable must not be group- or world-writable")
	}
	if !policy.checkOwner {
		return nil
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("executable ownership is unavailable")
	}
	for _, allowedOwner := range policy.allowedOwners {
		if stat.Uid == allowedOwner {
			return nil
		}
	}
	return fmt.Errorf("executable owner %d is not trusted", stat.Uid)
}

func sameOwnership(first, second os.FileInfo) bool {
	firstStat, firstOK := first.Sys().(*syscall.Stat_t)
	secondStat, secondOK := second.Sys().(*syscall.Stat_t)
	return firstOK && secondOK &&
		firstStat.Uid == secondStat.Uid &&
		firstStat.Gid == secondStat.Gid
}

func (registry Registry) lookup(id string) (source, error) {
	if registry.missing {
		return source{}, ErrRegistryMissing
	}
	if id == "" {
		return source{}, fmt.Errorf("%w: empty id", ErrSourceUnknown)
	}
	selected, ok := registry.sources[id]
	if !ok {
		return source{}, fmt.Errorf("%w: %q", ErrSourceUnknown, id)
	}
	return selected, nil
}

func parseTemperature(output []byte) (float32, error) {
	value := strings.TrimSpace(string(output))
	if value == "" {
		return 0, fmt.Errorf("%w: empty stdout", ErrInvalidOutput)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidOutput, value)
	}
	temperature := float32(math.Round(parsed*100) / 100)
	if math.IsNaN(float64(temperature)) || math.IsInf(float64(temperature), 0) {
		return 0, fmt.Errorf("%w: %q cannot be represented", ErrInvalidOutput, value)
	}
	return temperature, nil
}

type limitedCapture struct {
	buffer   bytes.Buffer
	limit    int
	tooLarge bool
}

func (capture *limitedCapture) Write(data []byte) (int, error) {
	originalLength := len(data)
	remaining := capture.limit + 1 - capture.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = capture.buffer.Write(data)
	}
	if capture.buffer.Len() > capture.limit || originalLength > remaining {
		capture.tooLarge = true
	}
	return originalLength, nil
}

func (capture *limitedCapture) Bytes() []byte {
	return capture.buffer.Bytes()
}

func (capture *limitedCapture) String() string {
	return capture.buffer.String()
}
