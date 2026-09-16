package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ConfabulousDev/confab/pkg/confabpath"
	"github.com/ConfabulousDev/confab/pkg/config"
	"github.com/ConfabulousDev/confab/pkg/logger"
	"github.com/ConfabulousDev/confab/pkg/process"
	"github.com/ConfabulousDev/confab/pkg/types"
)

const opencodePluginFileName = "confab-sync.ts"

type Opencode struct{}

var _ Provider = Opencode{}

func (Opencode) Name() string { return NameOpencode }

func (Opencode) CLIBinaryName() string { return "opencode" }

func (Opencode) SupportsCommitLinking() bool { return false }

func (p Opencode) ParseSessionHook(r io.Reader) (HookInput, error) {
	in, err := p.ReadSessionHookInput(r)
	if err != nil {
		return nil, err
	}
	return opencodeHookInputAdapter{inner: in}, nil
}

func (Opencode) WalkUpToRoot(sessionID string) (string, string, error) {
	return sessionID, "", nil
}

// ShouldSpawnForInput refuses subagent (non-root) OpenCode sessions so only the
// user-initiated root session spawns a daemon; CF-538 will capture subagents as
// sidechain files under the root. A session is a subagent when the plugin
// forwarded a parent session id (surfaced via an optional SessionParentID()
// accessor on the input — kept off the shared HookInput interface so Claude/
// Codex inputs need not implement it). Inputs without the accessor (or with an
// empty parent id) are treated as root.
func (Opencode) ShouldSpawnForInput(in HookInput) bool {
	if sp, ok := in.(interface{ SessionParentID() string }); ok && sp.SessionParentID() != "" {
		return false
	}
	return true
}

func (p Opencode) InstallHooks() (string, error) {
	source, err := currentOpencodePlugin()
	if err != nil {
		return "", err
	}
	pluginDir, err := p.PluginDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(pluginDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create plugin directory: %w", err)
	}
	pluginPath := filepath.Join(pluginDir, opencodePluginFileName)
	if err := os.WriteFile(pluginPath, []byte(source), 0644); err != nil {
		return "", fmt.Errorf("failed to write plugin: %w", err)
	}
	return pluginPath, nil
}

func (p Opencode) UninstallHooks() (string, error) {
	pluginDir, err := p.PluginDir()
	if err != nil {
		return "", err
	}
	pluginPath := filepath.Join(pluginDir, opencodePluginFileName)
	if err := os.Remove(pluginPath); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to remove plugin: %w", err)
	}
	return pluginPath, nil
}

func (p Opencode) IsHooksInstalled() (bool, error) {
	pluginDir, err := p.PluginDir()
	if err != nil {
		return false, err
	}
	pluginPath := filepath.Join(pluginDir, opencodePluginFileName)
	installed, err := os.ReadFile(pluginPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	expected, err := currentOpencodePlugin()
	if err != nil {
		return false, err
	}
	// 更新旧插件以及移动后的 EXE 绑定，避免 setup 因文件存在而跳过修复。
	return strings.ReplaceAll(string(installed), "\r\n", "\n") == strings.ReplaceAll(expected, "\r\n", "\n"), nil
}

func currentOpencodePlugin() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve confab executable for OpenCode plugin: %w", err)
	}
	return opencodePluginForExecutable(executable), nil
}

func opencodePluginForExecutable(executable string) string {
	// JSON 字符串也是合法的 TS 字符串；正确保留反斜杠、引号和 Unicode。
	encoded, _ := json.Marshal(executable) // Encoding a string cannot fail.
	return strings.Replace(opencodePluginSourceRaw, `const confabBinary = "confab"`, "const confabBinary = "+string(encoded), 1)
}

func (p Opencode) InstallSkills() error {
	stateDir, err := p.StateDir()
	if err != nil {
		return err
	}
	return config.ReconcileBundledSkills(stateDir, config.SkillProviderOpencode)
}

func (p Opencode) UninstallSkills() error {
	stateDir, err := p.StateDir()
	if err != nil {
		return err
	}
	return config.UninstallBundledSkills(stateDir)
}

func (p Opencode) IsSkillInstalled(name string) bool {
	stateDir, err := p.StateDir()
	if err != nil {
		return false
	}
	return config.IsBundledSkillInstalled(stateDir, name)
}

func (Opencode) WriteHookResponse(w io.Writer, _ bool, _ string) error {
	return nil
}

func (Opencode) InitTranscript(TranscriptRegistrar, string, string) error { return nil }

// opencodeListDescendantsTimeout bounds the per-cycle SQLite query that
// enumerates an OpenCode root's descendant sessions. Generous enough to
// survive heavy write contention (5s busy_timeout on the read connection
// plus a small margin); short enough that a wedged DB doesn't park the
// sync loop for the full 30s tick.
const opencodeListDescendantsTimeout = 10 * time.Second

// DiscoverDescendants enumerates every OpenCode subagent session under
// externalID via the local SQLite DB and registers each as a path-encoded
// sidechain file under the root's backend session (CF-538). The reg must
// implement OpencodeDescendantRegistrar (the daemon-supplied wrapper that
// drives child-collector goroutine spawn). When the assertion misses
// (forgotten daemon setter, or unit tests using a plain *FileTracker),
// logs a Warn once and returns nil — the production path requires the
// wrapper; the log surfaces a misconfiguration that would otherwise be
// silent.
//
// Per-tick semantics:
//  1. Resolve the DB path (CONFAB_OPENCODE_DB env, then $XDG_DATA_HOME,
//     then ~/.local/share). Provider is stateless: a fresh reader per
//     call costs one stat + open.
//  2. Recursive CTE walks session.parent_id descendants (capped at 1000
//     rows as a cycle defense).
//  3. For each descendant: derive the nested local materialized path
//     ~/.confab/opencode/<root>/children/<child>/messages.jsonl, then
//     call reg.RegisterOpencodeChild. The registrar handles capability
//     gating, file registration, and collector spawn — all idempotent.
//
// DB unavailable → log Warn, return nil (consistent with Codex's behavior
// when its state DB is missing). The daemon's sync cycle continues
// uninterrupted past a transient DB-absence.
func (Opencode) DiscoverDescendants(reg DescendantRegistrar, externalID string) error {
	oreg, ok := reg.(OpencodeDescendantRegistrar)
	if !ok {
		logger.Warn("OpenCode descendant discovery requires the daemon-supplied registrar; subagent capture disabled for session %s", externalID)
		return nil
	}

	dbPath, err := OpenCodeDBPath()
	if err != nil {
		logger.Warn("OpenCode DB path resolve failed: %v", err)
		return nil
	}
	reader := NewOpenCodeDBReader(dbPath)

	ctx, cancel := context.WithTimeout(context.Background(), opencodeListDescendantsTimeout)
	defer cancel()
	descendants, err := reader.ListDescendants(ctx, externalID)
	if err != nil {
		logger.Warn("OpenCode ListDescendants failed for %s: %v", externalID, err)
		return nil
	}

	for _, childID := range descendants {
		localPath, err := opencodeChildLocalPath(externalID, childID)
		if err != nil {
			logger.Warn("OpenCode child path derive failed for %s: %v", childID, err)
			continue
		}
		oreg.RegisterOpencodeChild(childID, localPath)
	}
	return nil
}

// opencodeChildLocalPath returns the per-child materialized JSONL path under
// ~/.confab/opencode/<root>/children/<child>/messages.jsonl. Nested under
// the root so a) cleanup tracks the root and b) two roots that
// (pathologically) reference the same child id never collide on disk.
//
// Backend file_name uses only the child id ("opencode/<child>/messages.jsonl");
// local path is decoupled, as TrackedFile.Path and TrackedFile.Name are
// independent.
func opencodeChildLocalPath(rootSessionID, childSessionID string) (string, error) {
	return confabpath.Subpath("opencode", rootSessionID, "children", childSessionID, "messages.jsonl")
}

// OpencodeChildBackendName returns the path-encoded backend file_name a
// daemon registrar should use when registering an OpenCode child file with
// the tracker. Forward slashes are load-bearing (the backend parses the
// path segments to resolve the child session id).
func OpencodeChildBackendName(childSessionID string) string {
	return path.Join("opencode", childSessionID, "messages.jsonl")
}

func (Opencode) DiscoverWorkflowFiles(WorkflowRegistrar, func(string) bool) (int, error) {
	return 0, nil
}

// AnnotateChunk sets first_user_message on the first transcript chunk so synced
// OpenCode sessions appear in the web session list (CF-540) — the backend's
// list query hides any session with neither a summary nor a first_user_message,
// and the CLI is the only source for those fields. OpenCode has no summary
// concept, so only first_user_message is set (mirroring Codex). The text is the
// first user message's first text part, trimmed and redacted (redact is
// nil-safe). A malformed materialized line degrades to "no message found"
// rather than failing the sync — we wrote these lines ourselves, so a parse
// error signals a collector bug worth a debug log, not a blocked upload.
func (Opencode) AnnotateChunk(c ChunkView, sentFirstUserMessage bool, redact func(string) string) AnnotationResult {
	var result AnnotationResult
	if sentFirstUserMessage || c.FileType() != "transcript" {
		return result
	}
	msg, err := ocFirstUserMessageText(c.Lines())
	if err != nil {
		logger.Debug("opencode: failed to extract first user message: %v", err)
		return result
	}
	if msg == "" {
		return result
	}
	if redact != nil {
		msg = redact(msg)
	}
	msg = TruncateUTF8(msg, types.MaxMetadataFieldLength/2)
	c.SetFirstUserMessage(msg)
	result.IncludedFirstUserMessage = true
	return result
}

func (Opencode) DefaultCWD(transcriptPath string) string {
	return filepath.Dir(transcriptPath)
}

// OnAlreadyRunning logs a warning that a parallel opencode process resumed
// the same session — a real edge case that confab's lifecycle model does
// not currently support reliably (CF-549, M2). The log goes to the confab
// log file only, not to opencode's stderr.
func (Opencode) OnAlreadyRunning(externalID string) {
	logger.Warn("opencode session %s has an existing daemon; multi-process resume is not supported and sync may be unreliable",
		externalID)
}

func (Opencode) FindParentPID() int {
	for pid, depth := os.Getppid(), 0; pid > 1 && depth < 5; pid, depth = process.ParentPID(pid), depth+1 {
		if opencodeProcessPattern.MatchString(process.Name(pid)) {
			return pid
		}
	}
	return 0
}

func (Opencode) IsProcess(pid int) bool {
	return opencodeProcessPattern.MatchString(process.Name(pid))
}

var opencodeProcessPattern = regexp.MustCompile(`(?i)\bopencode\b`)

func (p Opencode) StateDir() (string, error) {
	if envDir := os.Getenv("CONFAB_OPENCODE_CONFIG_DIR"); envDir != "" {
		return envDir, nil
	}
	// OpenCode resolves its config dir via xdg-basedir on every platform
	// (including Windows, where it reads %USERPROFILE%\.config, never
	// %APPDATA% — sst/opencode Global.Path = join(xdgConfig, "opencode")).
	// Honor $XDG_CONFIG_HOME the same way for parity.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "opencode"), nil
}

func (p Opencode) PluginDir() (string, error) {
	stateDir, err := p.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "plugins"), nil
}

func (p Opencode) ReadSessionHookInput(r io.Reader) (*types.OpenCodeHookInput, error) {
	return types.ReadHookInput(r, "OpenCode hook input", func(i *types.OpenCodeHookInput) string { return i.SessionID })
}

// opencodeScanTimeout bounds the SQLite reads ScanSessions/FindSessionByID
// issue for offline manual commands. Generous enough to survive a busy DB
// (5s busy_timeout on the read connection plus margin), bounded so a wedged DB
// fails the command cleanly rather than hanging.
const opencodeScanTimeout = 10 * time.Second

// ScanSessions enumerates OpenCode root sessions from the local SQLite DB for
// `confab list --provider opencode` (t6d5). Children are excluded (parent_id IS
// NULL), mirroring the daemon's root-only rule. Each row maps to a SessionInfo
// whose TITLE source is the first user message (OpenCode has no summary), read
// per-session via a bounded secondary query. The DB is opened read-only and
// honors CONFAB_OPENCODE_DB; a missing/unreadable DB returns a clear error.
//
// TranscriptPath is left empty here: OpenCode has no on-disk transcript, so the
// path is produced lazily by FindSessionByID (which materializes on demand).
func (Opencode) ScanSessions() ([]SessionInfo, error) {
	dbPath, err := OpenCodeDBPath()
	if err != nil {
		return nil, err
	}
	reader := NewOpenCodeDBReader(dbPath)

	ctx, cancel := context.WithTimeout(context.Background(), opencodeScanTimeout)
	defer cancel()

	roots, err := reader.ListRootSessions(ctx)
	if err != nil {
		return nil, err
	}

	sessions := make([]SessionInfo, 0, len(roots))
	for _, row := range roots {
		fum, err := reader.FirstUserMessageText(ctx, row.ID)
		if err != nil {
			// A per-session preview failure shouldn't drop the session from
			// the list; degrade to a blank TITLE.
			logger.Debug("opencode: first user message read failed for %s: %v", row.ID, err)
			fum = ""
		}
		sessions = append(sessions, SessionInfo{
			SessionID:        row.ID,
			ProjectPath:      row.Directory,
			ModTime:          time.Unix(row.TimeCreated, 0),
			FirstUserMessage: fum,
		})
	}
	return sessions, nil
}

// FindSessionByID resolves a full or partial OpenCode session id to its full
// ROOT id, materializes the root's transcript to
// ~/.confab/opencode/<root>/messages.jsonl, and returns (rootID, path). A
// descendant id resolves up to its root (consistent with the root+descendants
// save scope); descendants themselves are materialized + registered later by
// the save path (saveOpencodeDescendants). Prefix match mirrors the other
// providers: a unique prefix is required (ambiguous → error). Honors
// CONFAB_OPENCODE_DB; a missing/unreadable DB returns a clear error.
func (Opencode) FindSessionByID(partialID string) (string, string, error) {
	dbPath, err := OpenCodeDBPath()
	if err != nil {
		return "", "", err
	}
	reader := NewOpenCodeDBReader(dbPath)

	ctx, cancel := context.WithTimeout(context.Background(), opencodeScanTimeout)
	defer cancel()

	rootID, err := resolveOpencodeRoot(ctx, reader, partialID)
	if err != nil {
		return "", "", err
	}

	outputPath, err := confabpath.Subpath("opencode", rootID, "messages.jsonl")
	if err != nil {
		return "", "", err
	}
	if _, err := MaterializeOpenCodeSession(ctx, reader, rootID, outputPath, 0); err != nil {
		return "", "", fmt.Errorf("materialize opencode session %s: %w", rootID, err)
	}
	return rootID, outputPath, nil
}

// resolveOpencodeRoot resolves a (partial) session id to its full root id. It
// prefix-matches against ALL sessions (roots and descendants) so a user can
// pass any id from a tree; a descendant match walks up to its root via
// ResolveOpencodeRoot. A unique match is required.
func resolveOpencodeRoot(ctx context.Context, reader *OpenCodeDBReader, partialID string) (string, error) {
	matches, err := reader.MatchSessionIDs(ctx, partialID)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("session not found: %s", partialID)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous session ID '%s' matches %d sessions", partialID, len(matches))
	}
	return reader.ResolveOpencodeRoot(ctx, matches[0])
}

func (Opencode) ExtractMetadata([]string) SessionMetadata {
	return SessionMetadata{}
}

//go:embed plugins/confab-sync.ts
var opencodePluginSourceRaw string
