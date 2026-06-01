package main

import (
	"bytes"
	"compress/zlib"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed index.template.html pako_inflate.min.js
var templates embed.FS

type FileNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Size     int64       `json:"size_bytes"`
	MTime    time.Time   `json:"mtime"`
	Children []*FileNode `json:"children,omitempty"`
}

func buildTree(path string) (*FileNode, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	node := &FileNode{
		Name:  info.Name(),
		Size:  info.Size(),
		MTime: info.ModTime(),
	}
	if info.IsDir() {
		node.Type = "directory"
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil
		}
		for _, entry := range entries {
			childNode, err := buildTree(filepath.Join(path, entry.Name()))
			if err == nil {
				node.Children = append(node.Children, childNode)
			}
		}
	} else {
		node.Type = "file"
	}
	return node, nil
}

func printTree(node *FileNode, prefix string, isLast bool, buf *bytes.Buffer) {
	if node == nil {
		return
	}
	marker := "├── "
	if isLast {
		marker = "└── "
	}
	if node.Type == "directory" {
		buf.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, marker, node.Name))
	} else {
		buf.WriteString(fmt.Sprintf("%s%s%s (%d bytes)\n", prefix, marker, node.Name, node.Size))
	}
	nextPrefix := prefix
	if isLast {
		nextPrefix += "    "
	} else {
		nextPrefix += "│   "
	}
	for i, child := range node.Children {
		isChildLast := i == len(node.Children)-1
		printTree(child, nextPrefix, isChildLast, buf)
	}
}

func displayHelp() {
	fmt.Print(`
GoDir - A portable directory mapping utility built in Go.

Usage:
  godir.exe [OUTPUT MODE] [OPTIONS]
  godir.exe -dir <folderpath> [OUTPUT MODE] [OPTIONS]

Output Modes:
  --json, --tree, --index

Modifiers & Flags:
  -dir <path>     Target folder directory to map.
  -o <path>       Routes layout output to a custom location.
  -compress       Runs the data payload through a native Zlib compression pipeline.
  -force          Enables the utility to overwrite existing files.
  --help          Displays this help manual.
`)
}

func main() {
	targetDir := "."
	outputMode := ""
	compressOutput := false
	generateIndex := false
	forceOverwrite := false
	var outputFile string
	var modeFlagsPresent []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help":
			displayHelp()
			return
		case "--tree", "--json", "--index":
			if arg == "--index" {
				generateIndex = true
				outputMode = "json"
			} else {
				outputMode = strings.TrimPrefix(arg, "--")
			}
			modeFlagsPresent = append(modeFlagsPresent, arg)
		case "-compress":
			compressOutput = true
		case "-force":
			forceOverwrite = true
		case "-dir":
			if i+1 < len(args) {
				targetDir = args[i+1]
				i++
			}
		case "-o":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		default:
			if arg != "" {
				targetDir = arg
			}
		}
	}

	if len(modeFlagsPresent) > 1 {
		fmt.Fprintf(os.Stderr, "Error: Multiple output modes: %s\n", strings.Join(modeFlagsPresent, ", "))
		return
	}
	if outputMode == "" {
		outputMode = "json"
	}

	if generateIndex && outputFile == "" {
		outputFile = "."
	}

	var finalOutputPath string
	if outputFile != "" {
		absOutputPath, _ := filepath.Abs(outputFile)
		if generateIndex {
			fi, err := os.Stat(absOutputPath)
			// Force treat as directory, ignore provided filename
			if err == nil && fi.IsDir() {
				finalOutputPath = filepath.Join(absOutputPath, "index.html")
			} else {
				finalOutputPath = filepath.Join(filepath.Dir(absOutputPath), "index.html")
			}
		} else {
			finalOutputPath = absOutputPath
		}

		if _, err := os.Stat(finalOutputPath); err == nil && !forceOverwrite {
			fmt.Fprintf(os.Stderr, "Error: File '%s' already exists. Use -force to overwrite.\n", finalOutputPath)
			return
		}
	}

	absTargetDir, _ := filepath.Abs(targetDir)
	tree, err := buildTree(absTargetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	var corePayload []byte
	if outputMode == "tree" {
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("%s/\n", filepath.Base(absTargetDir)))
		for i, child := range tree.Children {
			printTree(child, "", i == len(tree.Children)-1, &buf)
		}
		corePayload = buf.Bytes()
	} else {
		if generateIndex {
			corePayload, _ = json.Marshal(tree)
		} else {
			corePayload, _ = json.MarshalIndent(tree, "", "  ")
		}
	}

	var finalOutputData []byte
	if compressOutput {
		var b bytes.Buffer
		w := zlib.NewWriter(&b)
		w.Write(corePayload)
		w.Close()
		data := b.Bytes()
		strNums := make([]string, len(data))
		for i, v := range data {
			strNums[i] = fmt.Sprintf("%d", v)
		}
		payload := strings.Join(strNums, ",")
		if generateIndex {
			finalOutputData = []byte("[" + payload + "]")
		} else {
			finalOutputData = []byte(payload)
		}
	} else {
		finalOutputData = corePayload
	}

	if finalOutputPath != "" {
		os.MkdirAll(filepath.Dir(finalOutputPath), 0755)

		if generateIndex {
			pakoContent := ""
			if compressOutput {
				pakoBytes, _ := templates.ReadFile("pako_inflate.min.js")
				pakoContent = string(pakoBytes)
			}
			templateData, _ := templates.ReadFile("index.template.html")
			htmlStr := strings.Replace(string(templateData), "{{PAKO_JS}}", pakoContent, 1)
			htmlStr = strings.Replace(htmlStr, "{{GODIR_DATA}}", string(finalOutputData), 1)

			os.WriteFile(finalOutputPath, []byte(htmlStr), 0644)
			fmt.Printf("Successfully deployed index.html to: %s\n", finalOutputPath)
		} else {
			os.WriteFile(finalOutputPath, finalOutputData, 0644)
			fmt.Printf("Successfully saved output to: %s\n", finalOutputPath)
		}
	} else {
		fmt.Println(string(finalOutputData))
	}
}
