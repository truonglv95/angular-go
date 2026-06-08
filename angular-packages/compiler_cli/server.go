package compiler_cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs/cachedvfs"
)

type RpcRequest struct {
	Id     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type RpcResponse struct {
	Id     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

const ProtocolVersion = 1

var ProtocolCapabilities = []string{
	"create_context",
	"build",
	"get_output",
	"get_hmr_update",
	"list_hmr_updates",
	"invalidate",
	"dispose_context",
	"link_file",
	"link_code",
	"shutdown",
}

type HelloResult struct {
	ProtocolVersion int      `json:"protocolVersion"`
	ServerName      string   `json:"serverName"`
	Capabilities    []string `json:"capabilities"`
}

type DiagnosticMessage struct {
	Category  string `json:"category"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	File      string `json:"file,omitempty"`
	Formatted string `json:"formatted,omitempty"`
}

type BuildResult struct {
	Outputs     []OutputFile        `json:"outputs"`
	Diagnostics []DiagnosticMessage `json:"diagnostics"`
	Status      int                 `json:"status"`
	PerfPhases  map[string]int64    `json:"perfPhases,omitempty"`
}

type BuildParams struct {
	Project         string `json:"project"`
	CompilationMode string `json:"compilationMode"`
	Hmr             bool   `json:"hmr"`
	ContextId       string `json:"contextId"`
	PreserveImports bool   `json:"preserveImports"`
}

// B#6 FIX: ContextState now tracks per-file invalidation instead of full cache clear.
type ContextState struct {
	Config    *ParsedConfiguration
	Host      compiler.CompilerHost
	NgProgram *ngtsc.NgtscProgram

	// H1 FIX: Cache last build outputs indexed by absolute path for get_output.
	outputs map[string]*OutputFile
	// Output manifest returned by build. Text is intentionally omitted so the
	// daemon does not serialize every compiled file on each build response.
	outputManifest []OutputFile
	lastPerfPhases map[string]int64

	// Map of files that have been invalidated since the last compilation.
	invalidatedFiles map[string]bool

	// B#5 FIX: RWMutex allows concurrent HMR reads while build holds write lock.
	mu sync.RWMutex
}

// buildMu serialises all build/create_context operations globally.
// The TypeScript Checker (LinkStore) is not goroutine-safe, so concurrent
// builds across different contexts would cause "concurrent map writes" panics.
var buildMu sync.Mutex

// B#5 FIX: Thread-safe contexts map protected by its own mutex.
var (
	contextsMu sync.RWMutex
	contexts   = make(map[string]*ContextState)
)

func getContext(id string) (*ContextState, bool) {
	contextsMu.RLock()
	defer contextsMu.RUnlock()
	s, ok := contexts[id]
	return s, ok
}

func setContext(id string, s *ContextState) {
	contextsMu.Lock()
	defer contextsMu.Unlock()
	contexts[id] = s
}

func deleteContext(id string) {
	contextsMu.Lock()
	defer contextsMu.Unlock()
	delete(contexts, id)
}

// getOrCreateContext atomically returns an existing context or creates a new one
// under the write lock, preventing the H7 race where two concurrent build requests
// for the same (not-yet-created) contextId both create separate ContextState objects.
func getOrCreateContext(params BuildParams) *ContextState {
	contextsMu.Lock()
	defer contextsMu.Unlock()
	if state, ok := contexts[params.ContextId]; ok {
		return state
	}
	config := ReadConfiguration(params.Project)
	config.CompilationMode = params.CompilationMode
	if config.CompilationMode == "" {
		config.CompilationMode = "global"
	}
	config.EnableHmr = params.Hmr
	if params.PreserveImports {
		EnablePreserveImports(config)
	}
	config.Write = false
	sys := newSystem(config.DiscardOutput)
	host := CreateCompilerHost(sys)
	state := &ContextState{Config: config, Host: host}
	contexts[params.ContextId] = state
	return state
}

type LinkParams struct {
	Path string `json:"path"`
}

// B#5 FIX: outputMu serialises writes to os.Stdout so concurrent goroutines
// don't interleave JSON lines.
var outputMu sync.Mutex

func sendResult(id int, result interface{}) {
	res := RpcResponse{Id: id, Result: result}
	outBytes, _ := json.Marshal(res)
	outputMu.Lock()
	os.Stdout.Write(outBytes)
	os.Stdout.WriteString("\n")
	outputMu.Unlock()
}

func sendError(id int, code string, message string) {
	res := RpcResponse{
		Id: id,
		Error: map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	outBytes, _ := json.Marshal(res)
	outputMu.Lock()
	os.Stdout.Write(outBytes)
	os.Stdout.WriteString("\n")
	outputMu.Unlock()
}

// RunServer reads JSON-RPC requests from stdin and dispatches each request in
// its own goroutine so that slow build operations never block fast HMR reads.
//
// C4 FIX: Track in-flight goroutines with a WaitGroup + context so they are
// cancelled/drained gracefully when stdin closes (EOF). Without this, goroutines
// that are mid-build would write to a closed stdout and panic.
//
// B#5 FIX: Concurrent request handling
// B#12 FIX: Use errors.Is(err, io.EOF) instead of string comparison
func RunServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Process all requests on a single goroutine to avoid concurrent-map-write
	// panics in the TypeScript Checker (LinkStore is not thread-safe).
	// This also guarantees that get_hmr_update always runs after build completes.
	reqCh := make(chan RpcRequest, 64)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for r := range reqCh {
			handleRequest(ctx, r)
		}
	}()

	decoder := json.NewDecoder(os.Stdin)
	for {
		var req RpcRequest
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			sendError(0, "ParseError", err.Error())
			continue
		}
		reqCh <- req
	}

	close(reqCh)
	cancel()
	wg.Wait()
}

func handleRequest(ctx context.Context, req RpcRequest) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "go-ngc server panic while handling %s: %v\n%s\n", req.Method, r, debug.Stack())
			sendError(req.Id, "InternalError", fmt.Sprintf("go-ngc server panic while handling %s: %v", req.Method, r))
		}
	}()

	// C4 FIX: Check if shutdown is in progress before doing heavy work.
	// Handlers that hold write locks will still finish — they just skip sending
	// the result if context is already done (stdout may be closed).
	select {
	case <-ctx.Done():
		return
	default:
	}

	switch req.Method {
	case "hello":
		sendResult(req.Id, HelloResult{
			ProtocolVersion: ProtocolVersion,
			ServerName:      "go-ngc",
			Capabilities:    ProtocolCapabilities,
		})

	case "create_context":
		// B#3 FIX: Pre-initialize context on the server so the first build
		// doesn't pay the ReadConfiguration cost mid-flight.
		var params struct {
			ContextId       string `json:"contextId"`
			Project         string `json:"project"`
			CompilationMode string `json:"compilationMode"`
			Hmr             bool   `json:"hmr"`
			PreserveImports *bool  `json:"preserveImports"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		// Serialise: ReadConfiguration touches the filesystem and Checker internals.
		buildMu.Lock()
		if _, ok := getContext(params.ContextId); !ok {
			config := ReadConfiguration(params.Project)
			if config.CompilationMode == "" {
				config.CompilationMode = params.CompilationMode
			}
			if config.CompilationMode == "" {
				config.CompilationMode = "global"
			}
			config.EnableHmr = params.Hmr
			if params.PreserveImports == nil || *params.PreserveImports {
				EnablePreserveImports(config)
			}
			config.Write = false
			sys := newSystem(config.DiscardOutput)
			host := CreateCompilerHost(sys)
			setContext(params.ContextId, &ContextState{Config: config, Host: host})
		}
		buildMu.Unlock()
		sendResult(req.Id, map[string]interface{}{"contextId": params.ContextId})

	case "build":
		var params BuildParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}

		// Acquire global build lock — TypeScript Checker is not goroutine-safe.
		// This also ensures build always runs after create_context completes.
		buildMu.Lock()
		defer buildMu.Unlock()

		// H7 FIX: Use contextsMu write lock to atomically check-and-create context.
		state := getOrCreateContext(params)

		// B#5 FIX: Write lock — build is exclusive; HMR reads must wait.
		state.mu.Lock()
		defer state.mu.Unlock()

		invalidated := state.invalidatedFiles
		state.invalidatedFiles = nil

		if state.NgProgram != nil && len(invalidated) == 0 && state.outputManifest != nil {
			sendResult(req.Id, BuildResult{
				Outputs:     state.outputManifest,
				Diagnostics: nil,
				Status:      int(tsc.ExitStatusSuccess),
				PerfPhases:  state.lastPerfPhases,
			})
			return
		}

		result := &PerformCompilationResult{
			Diagnostics: state.Config.Errors,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}
		if len(state.Config.Errors) == 0 {
			var ngProgram *ngtsc.NgtscProgram
			result, ngProgram = PerformCompilationWithHost(state.Config, state.Host, state.NgProgram, invalidated)
			state.NgProgram = ngProgram
		}

		// Deduplicate diagnostics
		{
			var uniqueDiags []*ast.Diagnostic
			seen := make(map[string]bool)
			for _, d := range result.Diagnostics {
				file := ""
				if d.File() != nil {
					file = d.File().FileName()
				}
				key := fmt.Sprintf("%s:%d:%d:%s", file, d.Code(), d.Pos(), d.String())
				if !seen[key] {
					seen[key] = true
					uniqueDiags = append(uniqueDiags, d)
				}
			}
			result.Diagnostics = uniqueDiags
		}

		var diags []DiagnosticMessage
		for _, d := range result.Diagnostics {
			file := ""
			if d.File() != nil {
				file = d.File().FileName()
			}
			category := "error"
			switch d.Category() {
			case 0: // CategoryWarning
				category = "warning"
			case 1: // CategoryError
				category = "error"
			case 2: // CategorySuggestion
				category = "suggestion"
			case 3: // CategoryMessage
				category = "message"
			}

			var sb strings.Builder
			compareOpts := tspath.ComparePathsOptions{
				CurrentDirectory:          state.Host.GetCurrentDirectory(),
				UseCaseSensitiveFileNames: true,
			}
			FormatDiagnosticEsbuildStyle(&sb, d, state.Config.Locale, compareOpts)
			formattedStr := sb.String()

			diags = append(diags, DiagnosticMessage{
				Category:  category,
				Code:      int(d.Code()),
				Message:   d.Localize(state.Config.Locale),
				File:      file,
				Formatted: formattedStr,
			})
		}

		// H1 FIX: Index outputs by path so get_output can serve them.
		if state.outputs == nil {
			state.outputs = make(map[string]*OutputFile)
		}
		for i := range result.Outputs {
			o := &result.Outputs[i]
			state.outputs[o.Path] = o
			absPath := o.Path
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(state.Host.GetCurrentDirectory(), absPath)
			}
			state.outputs[filepath.Clean(absPath)] = o
		}
		state.outputManifest = make([]OutputFile, len(result.Outputs))
		for i, output := range result.Outputs {
			state.outputManifest[i] = OutputFile{
				Path: output.Path,
				Hash: output.Hash,
				Kind: output.Kind,
			}
		}
		state.lastPerfPhases = result.PerfPhases

		sendResult(req.Id, BuildResult{
			Outputs:     state.outputManifest,
			Diagnostics: diags,
			Status:      int(result.Status),
			PerfPhases:  result.PerfPhases,
		})

	case "get_hmr_update":
		var params struct {
			ContextId   string `json:"contextId"`
			ComponentId string `json:"componentId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		// Serial dispatch guarantees this runs after build completes — no extra wait needed.
		state, ok := getContext(params.ContextId)
		if !ok || state.NgProgram == nil {
			sendError(req.Id, "InvalidContext", "Context not found or no program compiled yet")
			return
		}
		hmrCode := state.NgProgram.GetHmrUpdate(params.ComponentId)
		if hmrCode == "" {
			sendResult(req.Id, map[string]interface{}{"code": nil})
		} else {
			sendResult(req.Id, map[string]interface{}{"code": hmrCode})
		}

	case "list_hmr_updates":
		var params struct {
			ContextId string `json:"contextId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		state, ok := getContext(params.ContextId)
		if !ok || state.NgProgram == nil {
			sendError(req.Id, "InvalidContext", "Context not found or no program compiled yet")
			return
		}
		sendResult(req.Id, map[string]interface{}{"componentIds": state.NgProgram.GetHmrComponentIds()})

	case "invalidate":
		var params struct {
			ContextId string `json:"contextId"`
			File      string `json:"file"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		if state, ok := getContext(params.ContextId); ok {
			state.mu.Lock()
			if state.invalidatedFiles == nil {
				state.invalidatedFiles = make(map[string]bool)
			}
			state.invalidatedFiles[params.File] = true
			state.mu.Unlock()

			// B#6 FIX: Clear only the specific file's source cache entry.
			// The VFS layer doesn't support per-file eviction so we still
			// clear the full VFS cache — but source-level caching (which is
			// the expensive part) is now file-granular.
			if cc, ok := state.Host.(CacheClearable); ok {
				cc.ClearSourceFile(params.File)
			}
			// VFS-level cache still needs a full clear when a file changes.
			if cfs, ok := state.Host.FS().(*cachedvfs.FS); ok {
				cfs.ClearCache()
			}
		}
		sendResult(req.Id, map[string]interface{}{"success": true})

	case "get_output":
		// H1 FIX: Return in-memory output for a specific file path.
		// client.ts calls this to fetch compiled JS without going via the filesystem.
		var params struct {
			ContextId string `json:"contextId"`
			Path      string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		state, ok := getContext(params.ContextId)
		if !ok {
			sendResult(req.Id, nil)
			return
		}
		state.mu.RLock()
		cleanPath := filepath.Clean(params.Path)
		output := state.outputs[cleanPath]
		if output == nil {
			output = state.outputs[params.Path]
		}
		if output == nil && !filepath.IsAbs(cleanPath) {
			output = state.outputs[filepath.Join(state.Host.GetCurrentDirectory(), cleanPath)]
		}
		state.mu.RUnlock()
		if output == nil {
			sendResult(req.Id, nil)
		} else {
			sendResult(req.Id, output)
		}

	case "dispose_context":
		var params struct {
			ContextId string `json:"contextId"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		deleteContext(params.ContextId)
		sendResult(req.Id, map[string]interface{}{"success": true})

	case "link_file":
		var params LinkParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		contentBytes, err := os.ReadFile(params.Path)
		if err != nil {
			sendError(req.Id, "LinkError", err.Error())
			return
		}
		output, changed, err := LinkJavaScriptText(params.Path, string(contentBytes), core.ScriptKindTS)
		if err != nil {
			sendError(req.Id, "LinkError", err.Error())
		} else {
			sendResult(req.Id, map[string]interface{}{"code": output, "changed": changed})
		}

	case "link_code":
		var params struct {
			Code string `json:"code"`
			Id   string `json:"id"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.Id, "InvalidParams", err.Error())
			return
		}
		fileName := params.Id
		if !filepath.IsAbs(fileName) {
			fileName = filepath.Join(string(filepath.Separator), "virtual", params.Id)
		}
		output, changed, err := LinkJavaScriptText(fileName, params.Code, core.ScriptKindTS)
		if err != nil {
			sendError(req.Id, "LinkError", err.Error())
		} else {
			sendResult(req.Id, map[string]interface{}{"code": output, "changed": changed})
		}

	case "shutdown":
		sendResult(req.Id, map[string]interface{}{"success": true})
		// B#5 FIX: Flush stdout before exiting so the response is delivered.
		outputMu.Lock()
		os.Stdout.Sync()
		outputMu.Unlock()
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)

	default:
		sendError(req.Id, "MethodNotFound", fmt.Sprintf("Unknown method: %s", req.Method))
	}
}
