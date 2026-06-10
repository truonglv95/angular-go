package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Xliff struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:xliff:document:1.2 xliff"`
	Version string   `xml:"version,attr"`
	File    File     `xml:"file"`
}

type File struct {
	SourceLang string `xml:"source-language,attr"`
	TargetLang string `xml:"target-language,attr,omitempty"`
	Datatype   string   `xml:"datatype,attr"`
	Original   string   `xml:"original,attr"`
	Body       Body     `xml:"body"`
}

type Body struct {
	TransUnits []TransUnit `xml:"trans-unit"`
}

type TransUnit struct {
	Id       string  `xml:"id,attr"`
	Datatype string  `xml:"datatype,attr,omitempty"`
	Source   string  `xml:"source"`
	Target   *Target `xml:"target,omitempty"`
}

type Target struct {
	State   string `xml:"state,attr,omitempty"`
	Content string `xml:",innerxml"`
}

func parseXlf(path string) (map[string]*TransUnit, *Xliff, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil // Return empty if not exists
		}
		return nil, nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}

	var xliff Xliff
	if err := xml.Unmarshal(data, &xliff); err != nil {
		return nil, nil, err
	}

	dict := make(map[string]*TransUnit)
	for i := range xliff.File.Body.TransUnits {
		unit := &xliff.File.Body.TransUnits[i]
		dict[unit.Id] = unit
	}
	return dict, &xliff, nil
}

type ExtractedMsg struct {
	ID   string
	Text string
}

func extractFromFile(path string, msgsChan chan<- ExtractedMsg, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	data, err := os.ReadFile(path)
	if err != nil {
		errChan <- fmt.Errorf("failed to read %s: %w", path, err)
		return
	}

	localizeBytes := []byte("$localize`")
	if !bytes.Contains(data, localizeBytes) {
		return
	}

	i := 0
	for i < len(data) {
		idx := bytes.Index(data[i:], localizeBytes)
		if idx == -1 {
			break
		}

		i = i + idx + len(localizeBytes)
		startContentIdx := i
		braceDepth := 0
		inString := byte(0)
		isEscaped := false

		for i < len(data) {
			c := data[i]
			if isEscaped {
				isEscaped = false
				i++
				continue
			}
			if c == '\\' {
				isEscaped = true
				i++
				continue
			}
			if inString != 0 {
				if c == inString {
					inString = 0
				}
				i++
				continue
			}
			if braceDepth > 0 {
				if c == '"' || c == '\'' || c == '`' {
					inString = c
				} else if c == '{' {
					braceDepth++
				} else if c == '}' {
					braceDepth--
				}
				i++
				continue
			}
			if c == '$' && i+1 < len(data) && data[i+1] == '{' {
				braceDepth = 1
				i += 2
				continue
			}
			if c == '`' {
				break
			}
			i++
		}

		if i >= len(data) {
			break
		}

		endContentIdx := i
		content := string(data[startContentIdx:endContentIdx])
		i++

		id := ""
		origText := content
		if strings.HasPrefix(content, ":@@") {
			endIdIdx := strings.Index(content[3:], ":")
			if endIdIdx != -1 {
				id = content[3 : 3+endIdIdx]
				origText = content[3+endIdIdx+1:]
			}
		}

		if id != "" {
			msgsChan <- ExtractedMsg{ID: id, Text: origText}
		}
	}
}

func writeBaseXlf(extractedMap map[string]string, basePath string) {
	baseXliff := &Xliff{
		Version: "1.2",
		File: File{
			SourceLang: "en-US",
			Datatype:   "plaintext",
			Original:   "ng2.template",
		},
	}

	var units []TransUnit
	for id, text := range extractedMap {
		units = append(units, TransUnit{
			Id:       id,
			Datatype: "html",
			Source:   text,
		})
	}

	baseXliff.File.Body.TransUnits = units

	outBytes, err := xml.MarshalIndent(baseXliff, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal base XLF: %v", err)
	}

	finalBytes := append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\" ?>\n"), outBytes...)
	
	err = os.MkdirAll(filepath.Dir(basePath), 0755)
	if err != nil {
		log.Fatalf("Failed to create directory for base XLF: %v", err)
	}

	err = os.WriteFile(basePath, finalBytes, 0644)
	if err != nil {
		log.Fatalf("Failed to write base XLF: %v", err)
	}
}

func extractMode(dir string, basePath string, localesStr string) {
	start := time.Now()
	var wg sync.WaitGroup
	msgsChan := make(chan ExtractedMsg, 1000)
	errChan := make(chan error, 100)

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".js") {
			wg.Add(1)
			go extractFromFile(path, msgsChan, &wg, errChan)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Walk error: %v", err)
	}

	go func() {
		wg.Wait()
		close(msgsChan)
		close(errChan)
	}()

	extractedMap := make(map[string]string)
	for msg := range msgsChan {
		extractedMap[msg.ID] = msg.Text
	}

	for err := range errChan {
		log.Printf("Error extracting: %v", err)
	}

	// 1. Write the base messages.xlf
	if basePath == "" {
		basePath = "messages.xlf"
	}
	writeBaseXlf(extractedMap, basePath)

	// 2. Merge into locales
	if localesStr == "" {
		log.Printf("Extracted %d messages to %s in %v (No sync locales provided)", len(extractedMap), basePath, time.Since(start))
		return
	}

	locales := strings.Split(localesStr, ",")
	for _, mergePath := range locales {
		dict, existingXliff, err := parseXlf(mergePath)
		if err != nil {
			log.Fatalf("Failed to parse existing XLF %s: %v", mergePath, err)
		}

		if existingXliff == nil {
			existingXliff = &Xliff{
				Version: "1.2",
				File: File{
					SourceLang: "en-US",
					Datatype:   "plaintext",
					Original:   "ng2.template",
				},
			}
			dict = make(map[string]*TransUnit)
		}

		var newUnits []TransUnit
		for id, text := range extractedMap {
			if unit, exists := dict[id]; exists {
				unit.Source = text
				newUnits = append(newUnits, *unit)
			} else {
				newUnits = append(newUnits, TransUnit{
					Id:       id,
					Datatype: "html",
					Source:   text,
					Target:   &Target{State: "new", Content: text},
				})
			}
		}

		existingXliff.File.Body.TransUnits = newUnits

		outBytes, err := xml.MarshalIndent(existingXliff, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal XLF %s: %v", mergePath, err)
		}

		finalBytes := append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\" ?>\n"), outBytes...)
		
		err = os.MkdirAll(filepath.Dir(mergePath), 0755)
		if err != nil {
			log.Fatalf("Failed to create directory for output XLF %s: %v", mergePath, err)
		}

		err = os.WriteFile(mergePath, finalBytes, 0644)
		if err != nil {
			log.Fatalf("Failed to write output XLF %s: %v", mergePath, err)
		}
	}

	log.Printf("Extracted to %s and synced %d locales in %v", basePath, len(locales), time.Since(start))
}

func processFile(path string, dict map[string]*TransUnit, wg *sync.WaitGroup, errChan chan error) {
	defer wg.Done()
	// ... (Rest of replace logic, similar to before but reading dict[id].Target.Content)
	data, err := os.ReadFile(path)
	if err != nil {
		errChan <- fmt.Errorf("failed to read %s: %w", path, err)
		return
	}

	localizeBytes := []byte("$localize`")
	if !bytes.Contains(data, localizeBytes) {
		return
	}

	var buf bytes.Buffer
	buf.Grow(len(data))

	i := 0
	for i < len(data) {
		idx := bytes.Index(data[i:], localizeBytes)
		if idx == -1 {
			buf.Write(data[i:])
			break
		}

		buf.Write(data[i : i+idx])
		i = i + idx + len(localizeBytes)

		startContentIdx := i
		braceDepth := 0
		inString := byte(0)
		isEscaped := false

		for i < len(data) {
			c := data[i]
			if isEscaped {
				isEscaped = false
				i++
				continue
			}
			if c == '\\' {
				isEscaped = true
				i++
				continue
			}
			if inString != 0 {
				if c == inString {
					inString = 0
				}
				i++
				continue
			}
			if braceDepth > 0 {
				if c == '"' || c == '\'' || c == '`' {
					inString = c
				} else if c == '{' {
					braceDepth++
				} else if c == '}' {
					braceDepth--
				}
				i++
				continue
			}
			if c == '$' && i+1 < len(data) && data[i+1] == '{' {
				braceDepth = 1
				i += 2
				continue
			}
			if c == '`' {
				break
			}
			i++
		}

		if i >= len(data) {
			buf.Write(data[startContentIdx-len(localizeBytes):])
			break
		}

		endContentIdx := i
		content := string(data[startContentIdx:endContentIdx])
		i++

		id := ""
		if strings.HasPrefix(content, ":@@") {
			endIdIdx := strings.Index(content[3:], ":")
			if endIdIdx != -1 {
				id = content[3 : 3+endIdIdx]
			}
		}

		if unit, exists := dict[id]; exists && unit.Target != nil {
			escaped := strings.ReplaceAll(unit.Target.Content, "\"", "\\\"")
			escaped = strings.ReplaceAll(escaped, "\n", "\\n")
			buf.WriteString(`"` + escaped + `"`)
		} else {
			origText := content
			if id != "" {
				origText = content[3+len(id)+1:]
			}
			escapedOrig := strings.ReplaceAll(origText, "\"", "\\\"")
			escapedOrig = strings.ReplaceAll(escapedOrig, "\n", "\\n")
			buf.WriteString(`"` + escapedOrig + `"`)
		}
	}

	err = os.WriteFile(path, buf.Bytes(), 0644)
	if err != nil {
		errChan <- fmt.Errorf("failed to write %s: %w", path, err)
	}
}

func replaceMode(dir string, localeStr string) {
	start := time.Now()
	locales := strings.Split(localeStr, ",")
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	for _, l := range locales {
		parts := strings.SplitN(l, ":", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid locale string: %s", l)
		}
		lang := parts[0]
		xlfPath := parts[1]

		dict, _, err := parseXlf(xlfPath)
		if err != nil {
			log.Fatalf("Failed to parse XLF %s: %v", xlfPath, err)
		}

		outDir := filepath.Join(dir, lang)
		err = os.MkdirAll(outDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create outDir: %v", err)
		}

		err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(dir, path)
			if err != nil || strings.HasPrefix(relPath, lang) {
				return nil
			}
			isLangDir := false
			for _, loc := range locales {
				lparts := strings.SplitN(loc, ":", 2)
				if strings.HasPrefix(relPath, lparts[0]) {
					isLangDir = true
					break
				}
			}
			if isLangDir {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			outPath := filepath.Join(outDir, relPath)
			if info.IsDir() {
				return os.MkdirAll(outPath, 0755)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = os.WriteFile(outPath, data, info.Mode())
			if err != nil {
				return err
			}

			if strings.HasSuffix(info.Name(), ".js") {
				wg.Add(1)
				go processFile(outPath, dict, &wg, errChan)
			}
			return nil
		})

		if err != nil {
			log.Fatalf("Walk error: %v", err)
		}
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		log.Printf("Error: %v", err)
	}

	log.Printf("Localized %d locales in %v", len(locales), time.Since(start))
}

func main() {
	mode := flag.String("mode", "replace", "Mode: 'replace' or 'extract'")
	dir := flag.String("dir", "", "Directory containing JS files")
	outFile := flag.String("outFile", "", "Base XLF output file path for extract mode")
	localeStr := flag.String("locales", "", "replace mode: vi:path.xlf, extract mode: comma-separated list of target .xlf files to sync")
	flag.Parse()

	if *dir == "" {
		log.Fatalf("Usage: go-localize --mode=<mode> --dir=<dir> [--outFile=<base.xlf>] [--locales=<locales/path>]")
	}

	if *mode == "extract" {
		extractMode(*dir, *outFile, *localeStr)
	} else {
		replaceMode(*dir, *localeStr)
	}
}
