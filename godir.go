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

//go:embed index.template.html index-pako.template.html
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
	helpText := `
GoDir - A portable directory mapping utility built in Go.

Usage:
  godir.exe [OUTPUT MODE] [OPTIONS]
  godir.exe -dir <folderpath> [OUTPUT MODE] [OPTIONS]

Output Modes (Mutually Exclusive - choose only one):
  --json             Outputs directory structure as an indented JSON schema payload (Default).
  --tree             Outputs a visual, classic ASCII terminal file tree blueprint.
  --js               Outputs a valid JavaScript asset statement (e.g., const godir = { ... };).
  --index            Outputs a valid 'godir.js' file and embeds a template HTML file to the 
                     output directory as 'index.html'. (Automatically infers JavaScript formatting).

Modifiers & Flags:
  -dir <path>        Target folder directory to map. Supports relative and absolute paths.
                     Defaults to current directory.
  -o <filename/path> Routes layout output to a custom location. Optional if --index is used.
  --compress         Runs the data payload through a native Zlib compression pipeline.
  --overwrite        Enables the utility to overwrite an existing 'index.html' file when
                     using the --index output mode.
  --help             Displays this CLI usage instructions manual.
`
	fmt.Print(helpText)
}

func main() {
	targetDir := "."
	outputMode := ""
	compressOutput := false
	generateIndex := false
	allowOverwrite := false
	var outputFile string

	var modeFlagsPresent []string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--help" || arg == "-h" {
			displayHelp()
			return
		} else if arg == "--tree" {
			outputMode = "tree"
			modeFlagsPresent = append(modeFlagsPresent, "--tree")
		} else if arg == "--json" {
			outputMode = "json"
			modeFlagsPresent = append(modeFlagsPresent, "--json")
		} else if arg == "--js" {
			outputMode = "js"
			modeFlagsPresent = append(modeFlagsPresent, "--js")
		} else if arg == "--index" {
			generateIndex = true
			outputMode = "js"
			modeFlagsPresent = append(modeFlagsPresent, "--index")
		} else if arg == "--compress" {
			compressOutput = true
		} else if arg == "--overwrite" {
			allowOverwrite = true
		} else if arg == "-dir" {
			if i+1 < len(args) {
				targetDir = args[i+1]
				i++
			} else {
				fmt.Fprintln(os.Stderr, "Error: -dir requires a folder path parameter.")
				return
			}
		} else if arg == "-o" {
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			} else {
				fmt.Fprintln(os.Stderr, "Error: -o requires a filename or full path.")
				return
			}
		} else if arg != "" {
			targetDir = arg
		}
	}

	if len(modeFlagsPresent) > 1 {
		fmt.Fprintf(os.Stderr, "Error: The output mode flags %s cannot be used together in the same command. Please choose exactly one output mode flag.\n", strings.Join(modeFlagsPresent, ", "))
		return
	}

	if outputMode == "" {
		outputMode = "json"
	}

	if generateIndex && outputFile == "" {
		outputFile = "."
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving target directory path: %v\n", err)
		return
	}

	tree, err := buildTree(absTargetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing path: %v\n", err)
		return
	}

	var corePayload []byte

	if outputMode == "tree" {
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("%s/\n", filepath.Base(absTargetDir)))
		for i, child := range tree.Children {
			isLast := i == len(tree.Children)-1
			printTree(child, "", isLast, &buf)
		}
		corePayload = buf.Bytes()
	} else {
		jsonData, _ := json.MarshalIndent(tree, "", "  ")
		corePayload = jsonData
	}

	var finalOutputData []byte

	if compressOutput {
		var compressedBuf bytes.Buffer
		zlibWriter := zlib.NewWriter(&compressedBuf)
		_, _ = zlibWriter.Write(corePayload)
		zlibWriter.Close()
		compressedBytes := compressedBuf.Bytes()

		if outputMode == "js" {
			var strNumbers []string
			for _, b := range compressedBytes {
				strNumbers = append(strNumbers, fmt.Sprintf("%d", b))
			}
			jsCode := fmt.Sprintf("const godir = [%s];\n", strings.Join(strNumbers, ","))
			finalOutputData = []byte(jsCode)
		} else {
			var strNumbers []string
			for _, b := range compressedBytes {
				strNumbers = append(strNumbers, fmt.Sprintf("%d", b))
			}
			finalOutputData = []byte(strings.Join(strNumbers, ","))
		}
	} else {
		if outputMode == "js" {
			jsCode := fmt.Sprintf("const godir = %s;\n", string(corePayload))
			finalOutputData = []byte(jsCode)
		} else {
			finalOutputData = corePayload
		}
	}

	if outputFile != "" {
		absOutputPath, err := filepath.Abs(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving output path: %v\n", err)
			return
		}

		var outputDir string
		if generateIndex {
			fi, err := os.Stat(absOutputPath)
			if err == nil && fi.IsDir() {
				outputDir = absOutputPath
			} else {
				if strings.HasSuffix(outputFile, string(filepath.Separator)) || outputFile == "." {
					outputDir = absOutputPath
				} else {
					outputDir = filepath.Dir(absOutputPath)
				}
			}
			absOutputPath = filepath.Join(outputDir, "godir.js")
		} else {
			outputDir = filepath.Dir(absOutputPath)
		}

		if generateIndex {
			dstIndexPath := filepath.Join(outputDir, "index.html")
			if _, err := os.Stat(dstIndexPath); err == nil {
				if !allowOverwrite {
					fmt.Fprintf(os.Stderr, "Error: 'index.html' already exists at destination location (%s).\nExecution aborted. No files were altered. Use the --overwrite flag to bypass this restriction.\n", dstIndexPath)
					return
				}
			}
		}

		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory structure: %v\n", err)
			return
		}

		err = os.WriteFile(absOutputPath, finalOutputData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing data file: %v\n", err)
			return
		}
		fmt.Printf("Successfully saved layout output to: %s\n", absOutputPath)

		if generateIndex {
			templateName := "index.template.html"
			if compressOutput {
				templateName = "index-pako.template.html"
			}

			templateData, err := templates.ReadFile(templateName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading embedded template: %v\n", err)
				return
			}

			dstIndexPath := filepath.Join(outputDir, "index.html")
			err = os.WriteFile(dstIndexPath, templateData, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing embedded template file: %v\n", err)
				return
			}
			fmt.Printf("Successfully deployed embedded template and created layout view at: %s\n", dstIndexPath)
		}
	} else {
		fmt.Println(string(finalOutputData))
	}
}
