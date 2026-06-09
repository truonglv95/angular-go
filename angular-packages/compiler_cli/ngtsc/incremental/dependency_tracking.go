package incremental

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/microsoft/typescript-go/internal/tspath"
)

type FileDependencyGraph struct {
	mu                  sync.RWMutex
	depsByFile          map[string]map[string]bool
	reverseDeps         map[string]map[string]bool
	resourceDepsByFile  map[string]map[string]bool
	reverseResourceDeps map[string]map[string]bool
	failedAnalysisFiles map[string]bool
}

func NewFileDependencyGraph() *FileDependencyGraph {
	return &FileDependencyGraph{
		depsByFile:          make(map[string]map[string]bool),
		reverseDeps:         make(map[string]map[string]bool),
		resourceDepsByFile:  make(map[string]map[string]bool),
		reverseResourceDeps: make(map[string]map[string]bool),
		failedAnalysisFiles: make(map[string]bool),
	}
}

func canonicalizePath(path string) string {
	if path == "" {
		return ""
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return tspath.GetCanonicalFileName(path, false)
	}
	return tspath.GetCanonicalFileName(path, true)
}

func getPath(v any) string {
	if v == nil {
		return ""
	}
	var path string
	if s, ok := v.(string); ok {
		path = s
	} else if sf, ok := v.(interface{ FileName() string }); ok {
		path = sf.FileName()
	} else {
		path = fmt.Sprintf("%v", v)
	}
	return canonicalizePath(path)
}

func (recv *FileDependencyGraph) ensureInitialized() {
	if recv.depsByFile == nil {
		recv.depsByFile = make(map[string]map[string]bool)
	}
	if recv.reverseDeps == nil {
		recv.reverseDeps = make(map[string]map[string]bool)
	}
	if recv.resourceDepsByFile == nil {
		recv.resourceDepsByFile = make(map[string]map[string]bool)
	}
	if recv.reverseResourceDeps == nil {
		recv.reverseResourceDeps = make(map[string]map[string]bool)
	}
	if recv.failedAnalysisFiles == nil {
		recv.failedAnalysisFiles = make(map[string]bool)
	}
}

func (recv *FileDependencyGraph) AddDependency(from any, on any) any {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	recv.ensureInitialized()

	fromPath := getPath(from)
	onPath := getPath(on)
	if fromPath == "" || onPath == "" {
		return nil
	}

	if recv.depsByFile[fromPath] == nil {
		recv.depsByFile[fromPath] = make(map[string]bool)
	}
	recv.depsByFile[fromPath][onPath] = true

	if recv.reverseDeps[onPath] == nil {
		recv.reverseDeps[onPath] = make(map[string]bool)
	}
	recv.reverseDeps[onPath][fromPath] = true

	return nil
}

func (recv *FileDependencyGraph) AddResourceDependency(from any, resource string) any {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	recv.ensureInitialized()

	fromPath := getPath(from)
	resourcePath := canonicalizePath(resource)
	if fromPath == "" || resourcePath == "" {
		return nil
	}

	if recv.resourceDepsByFile[fromPath] == nil {
		recv.resourceDepsByFile[fromPath] = make(map[string]bool)
	}
	recv.resourceDepsByFile[fromPath][resourcePath] = true

	if recv.reverseResourceDeps[resourcePath] == nil {
		recv.reverseResourceDeps[resourcePath] = make(map[string]bool)
	}
	recv.reverseResourceDeps[resourcePath][fromPath] = true

	return nil
}

func (recv *FileDependencyGraph) RecordDependencyAnalysisFailure(file any) any {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	recv.ensureInitialized()

	filePath := getPath(file)
	if filePath != "" {
		recv.failedAnalysisFiles[filePath] = true
	}
	return nil
}

func (recv *FileDependencyGraph) GetResourceDependencies(from any) []string {
	recv.mu.RLock()
	defer recv.mu.RUnlock()
	if recv.resourceDepsByFile == nil {
		return nil
	}

	fromPath := getPath(from)
	if fromPath == "" {
		return nil
	}

	resMap := recv.resourceDepsByFile[fromPath]
	if len(resMap) == 0 {
		return nil
	}

	res := make([]string, 0, len(resMap))
	for k := range resMap {
		res = append(res, k)
	}
	return res
}

func (recv *FileDependencyGraph) UpdateWithPhysicalChanges(previous FileDependencyGraph, changedTsPaths map[string]bool, deletedTsPaths map[string]bool, changedResources map[string]bool) map[string]bool {
	recv.mu.Lock()
	defer recv.mu.Unlock()
	recv.ensureInitialized()

	previous.mu.RLock()
	for k, v := range previous.depsByFile {
		recv.depsByFile[k] = make(map[string]bool)
		for k2, v2 := range v {
			recv.depsByFile[k][k2] = v2
		}
	}
	for k, v := range previous.reverseDeps {
		recv.reverseDeps[k] = make(map[string]bool)
		for k2, v2 := range v {
			recv.reverseDeps[k][k2] = v2
		}
	}
	for k, v := range previous.resourceDepsByFile {
		recv.resourceDepsByFile[k] = make(map[string]bool)
		for k2, v2 := range v {
			recv.resourceDepsByFile[k][k2] = v2
		}
	}
	for k, v := range previous.reverseResourceDeps {
		recv.reverseResourceDeps[k] = make(map[string]bool)
		for k2, v2 := range v {
			recv.reverseResourceDeps[k][k2] = v2
		}
	}
	for k, v := range previous.failedAnalysisFiles {
		recv.failedAnalysisFiles[k] = v
	}
	previous.mu.RUnlock()

	affected := make(map[string]bool)
	var queue []string

	for path := range changedTsPaths {
		affected[path] = true
		queue = append(queue, path)
	}

	for path := range deletedTsPaths {
		affected[path] = true
		queue = append(queue, path)
		delete(recv.depsByFile, path)
		delete(recv.resourceDepsByFile, path)
		for k, v := range recv.reverseDeps {
			delete(v, path)
			if len(v) == 0 {
				delete(recv.reverseDeps, k)
			}
		}
		for k, v := range recv.reverseResourceDeps {
			delete(v, path)
			if len(v) == 0 {
				delete(recv.reverseResourceDeps, k)
			}
		}
	}

	for res := range changedResources {
		if files, ok := recv.reverseResourceDeps[res]; ok {
			for f := range files {
				if !affected[f] {
					affected[f] = true
					queue = append(queue, f)
				}
			}
		}
	}

	for f := range recv.failedAnalysisFiles {
		if !affected[f] {
			affected[f] = true
			queue = append(queue, f)
		}
	}
	recv.failedAnalysisFiles = make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if deps, ok := recv.reverseDeps[current]; ok {
			for dep := range deps {
				if !affected[dep] {
					affected[dep] = true
					queue = append(queue, dep)
				}
			}
		}
	}

	return affected
}
