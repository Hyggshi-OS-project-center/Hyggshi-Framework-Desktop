package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/hyggshi/hfd"
)

//go:embed all:frontend/dist
var distFS embed.FS

// assets — the root served at hfd://app/ (strip the embed prefix).
func mustSub() fs.FS {
	sub, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		log.Fatal("hfd-code: embed assets: ", err)
	}
	return sub
}

// ---- Typed DTOs — the source of the bindings codegen ------------------------
// JSON field names are the contract with the frontend, not the Go field names.

type OpenFileArgs struct{}

type OpenFileResult struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Canceled bool   `json:"canceled"`
}

type SaveArgs struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type SaveResult struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

type SetTitleArgs struct {
	Title string `json:"title"`
}

type SetTitleResult struct {
	Ok bool `json:"ok"`
}

const maxFileBytes = 4 << 20 // 4 MB — the demo refuses larger files to keep the UI smooth

func errTooLarge(path string) error {
	if path == "" {
		path = "file"
	}
	return &tooLargeError{path: path}
}

type tooLargeError struct{ path string }

func (e *tooLargeError) Error() string {
	return e.path + " is larger than the 4 MB demo limit"
}

func main() {
	hfd.BindTyped("OpenFileDialog", func(_ OpenFileArgs) (OpenFileResult, error) {
		path, ok, err := hfd.OpenFileDialog(hfd.FileDialogOptions{
			Title:    "Open file",
			StartDir: homeOrDot(),
			Filters: []hfd.FileFilter{
				{Name: "Text & code", Patterns: []string{"*.go", "*.js", "*.mjs", "*.ts", "*.tsx", "*.json", "*.css", "*.html", "*.md", "*.txt", "*.py", "*.rs", "*.c", "*.h", "*.java"}},
				{Name: "All files", Patterns: []string{"*"}},
			},
		})
		if err != nil || !ok {
			return OpenFileResult{Canceled: true}, err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return OpenFileResult{}, err
		}
		if len(b) > maxFileBytes {
			return OpenFileResult{}, errTooLarge(path)
		}
		_ = hfd.SetTitle(titleFor(path, false))
		return OpenFileResult{Path: path, Content: string(b)}, nil
	})

	hfd.BindTyped("SaveFile", func(a SaveArgs) (SaveResult, error) {
		path := a.Path
		if path == "" {
			p, ok, err := hfd.SaveFileDialog(hfd.FileDialogOptions{
				Title:       "Save file",
				InitialName: suggestedName(path),
			})
			if err != nil || !ok {
				return SaveResult{Path: path}, err
			}
			path = p
		}
		if len(a.Text) > maxFileBytes {
			return SaveResult{Path: path}, errTooLarge(path)
		}
		if err := os.WriteFile(path, []byte(a.Text), 0o644); err != nil {
			return SaveResult{Path: path}, err
		}
		_ = hfd.SetTitle(titleFor(path, false))
		return SaveResult{Path: path, Bytes: len(a.Text)}, nil
	})

	hfd.BindTyped("SetTitle", func(a SetTitleArgs) (SetTitleResult, error) {
		return SetTitleResult{Ok: hfd.SetTitle(a.Title) == nil}, nil
	})

	// About → native message box, handled straight in Go (no JS round-trip).
	hfd.Bind("AboutApp", func(_ string) (any, error) {
		_, err := hfd.ShowMessage(hfd.MessageDialogOptions{
			Title: "About",
			Message: "HFD Code Editor — a demo of Hyggshi Framework Desktop\n" +
				"Go runtime + GTK/WebKitGTK; UI is plain HTML/CSS/JS.\n" +
				"100% written from scratch — no Wails/Tauri/Electron.",
			Level: hfd.MessageInfo,
		})
		return nil, err
	})

	// Menu bar — clicks arrive in JS as the "menu" event { action }.
	// "help-about" is handled in Go: a native message box, no JS round-trip.

	if err := hfd.Run(hfd.Options{
		Title:  titleFor("", false),
		Width:  1000,
		Height: 720,
		Assets: mustSub(),
		MenuLabel: "HFD Code",
		Menu: []hfd.MenuItem{
			{Label: "New", Action: "file-new"},
			{Label: "Open…", Action: "file-open"},
			{Label: "Save", Action: "file-save"},
			{Label: "Save As…", Action: "file-saveas"},
			{Label: "About", Action: "help-about"},
		},
	}); err != nil {
		log.Println("hfd:", err)
		os.Exit(1)
	}
}

func homeOrDot() string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "."
	}
	return h
}

func suggestedName(path string) string {
	if path == "" {
		return "untitled.txt"
	}
	return filepath.Base(path)
}

func titleFor(path string, dirty bool) string {
	base := "untitled"
	if path != "" {
		base = filepath.Base(path)
	}
	dot := ""
	if dirty {
		dot = "• "
	}
	return dot + base + " — HFD Code"
}
