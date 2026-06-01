# GoDir

A lightweight, portable directory mapping utility written in Go. It allows you to scan file system structures and export them in various formats—including JSON, ASCII trees, and ready-to-use JavaScript data files for web applications.

This tool was made with the assistance of **Gemini**.

---

## Features

* **Flexible Output:** Generate directory trees as standard JSON, visual ASCII trees, or JavaScript variables (`const godir = ...`).
* **Web Integration:** Use the `--index` flag to generate a `godir.js` file alongside a companion `index.html` template for an instant, interactive web-based file explorer.
* **Compression:** Built-in Zlib compression support for handling large file structures efficiently.
* **Zero Dependencies:** Designed as a single-binary utility for portability.

---

## Usage

### Basic Commands

* **Default (JSON):** `godir.exe`
* **Visual Tree:** `godir.exe --tree`
* **Custom Directory:** `godir.exe -dir /path/to/folder --json`

### Web Export Mode

To create a web-ready file viewer:
`godir.exe -dir ./my-files --index --overwrite`

---

## Options

| Flag | Description |
| --- | --- |
| `--json` | Outputs directory structure as an indented JSON schema (Default). |
| `--tree` | Outputs a visual, classic ASCII terminal file tree. |
| `--js` | Outputs a valid JavaScript constant assignment. |
| `--index` | Generates `godir.js` and an `index.html` template. |
| `-dir <path>` | Specifies the target directory to map (defaults to current). |
| `-o <file>` | Routes output to a specific file path. |
| `--compress` | Compresses the data payload using Zlib. |
| `--overwrite` | Allows overwriting of existing index files. |
| `--help` | Displays the full help manual. |

---

## License

This project is licensed under the [**MIT License**](./LICENSE).