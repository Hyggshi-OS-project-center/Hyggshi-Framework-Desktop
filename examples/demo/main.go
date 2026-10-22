package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hyggshi/hfd"
)

//go:embed all:frontend/dist
var distFS embed.FS

// ---- Typed DTOs — this is the source of the bindings.d.ts codegen table ----
// The JSON field names are the contract with the frontend, not the Go field names.

type GreetArgs struct {
	Name string `json:"name"`
}

type GreetResult struct {
	Message string `json:"message"`
}

type SysInfoArgs struct{}

type SysInfoResult struct {
	Hostname string   `json:"hostname"`
	OS       string   `json:"os"`
	Arch     string   `json:"arch"`
	CPUs     int      `json:"cpus"`
	GoVer    string   `json:"goVersion"`
	Tags     []string `json:"tags"`
}

type StartTickArgs struct {
	Times int `json:"times"` // JS says how many ticks to fire
}

type StartTickResult struct {
	Started bool `json:"started"`
}

// ---- Phase 4: native API typed -------------------------------------------

type PickFileArgs struct {
	Title string `json:"title"`
}

type PickFileResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

type SaveTextArgs struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type SaveTextResult struct {
	Path     string `json:"path"`
	Bytes    int    `json:"bytes"`
	Canceled bool   `json:"canceled"`
}

type SetTextArgs struct {
	Text string `json:"text"`
}

type SetTextResult struct {
	Saved bool `json:"saved"`
}

type NotifyArgs struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type NotifyResult struct {
	Sent bool   `json:"sent"`
	Err  string `json:"err,omitempty"`
}

type EchoResult struct {
	Text string `json:"text"`
}

// App holds the methods exposed over IPC via BindTyped.
type App struct{}

func (a *App) Greet(args GreetArgs) (GreetResult, error) {
	if args.Name == "" {
		args.Name = "there"
	}
	return GreetResult{
		Message: "Hello " + args.Name + " — answered straight from the HFD Go runtime!",
	}, nil
}

func (a *App) SysInfo(_ SysInfoArgs) (SysInfoResult, error) {
	host, _ := os.Hostname()
	return SysInfoResult{
		Hostname: host,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
		GoVer:    runtime.Version(),
		Tags:     []string{"no-wails", "no-electron", "pure-cgo"},
	}, nil
}

func (a *App) StartTick(args StartTickArgs) (StartTickResult, error) {
	if args.Times <= 0 {
		args.Times = 5
	}
	go func(n int) {
		for i := 1; i <= n; i++ {
			if err := hfd.Emit("tick", map[string]any{"count": i}); err != nil {
				return // app closed
			}
			time.Sleep(time.Second)
		}
		_ = hfd.Emit("tick-done", map[string]string{"note": fmt.Sprintf("completed %d ticks", n)})
	}(args.Times)
	return StartTickResult{Started: true}, nil
}

// ---- Phase 4: native API implementations --------------------------------

func (a *App) PickFile(args PickFileArgs) (PickFileResult, error) {
	if args.Title == "" {
		args.Title = "Pick a file"
	}
	path, ok, err := hfd.OpenFileDialog(hfd.FileDialogOptions{
		Title:    args.Title,
		StartDir: ".",
		Filters: []hfd.FileFilter{
			{Name: "All files", Patterns: []string{"*"}},
			{Name: "Text", Patterns: []string{"*.txt", "*.md", "*.go"}},
		},
	})
	if err != nil {
		return PickFileResult{}, err
	}
	return PickFileResult{Path: path, Canceled: !ok}, nil
}

func (a *App) SaveText(args SaveTextArgs) (SaveTextResult, error) {
	if args.Name == "" {
		args.Name = "hfd-note.txt"
	}
	path, ok, err := hfd.SaveFileDialog(hfd.FileDialogOptions{
		Title:       "Save file",
		InitialName: args.Name,
	})
	if err != nil {
		return SaveTextResult{}, err
	}
	if !ok {
		return SaveTextResult{Canceled: true}, nil
	}
	if err := os.WriteFile(path, []byte(args.Text), 0o644); err != nil {
		return SaveTextResult{}, err
	}
	return SaveTextResult{Path: path, Bytes: len(args.Text)}, nil
}

func (a *App) ClipboardSet(args SetTextArgs) (SetTextResult, error) {
	if err := hfd.SetClipboard(args.Text); err != nil {
		return SetTextResult{}, err
	}
	return SetTextResult{Saved: true}, nil
}

func (a *App) ClipboardGet(_ SetTextArgs) (EchoResult, error) {
	text, err := hfd.GetClipboard()
	if err != nil {
		return EchoResult{}, err
	}
	return EchoResult{Text: text}, nil
}

func (a *App) SendNotification(args NotifyArgs) (NotifyResult, error) {
	if args.Title == "" {
		args.Title = "HFD"
	}
	if err := hfd.Notify(args.Title, args.Body); err != nil {
		return NotifyResult{Sent: false, Err: err.Error()}, nil // do not fail the IPC
	}
	return NotifyResult{Sent: true}, nil
}

func (a *App) AskConfirm(_ SetTextArgs) (EchoResult, error) {
	ok, err := hfd.ShowMessage(hfd.MessageDialogOptions{
		Title:   "HFD",
		Message: "We have reached Phase 4 (native dialogs)! Fun, right?",
		Level:   hfd.MessageQuestion,
	})
	if err != nil {
		return EchoResult{}, err
	}
	if ok {
		return EchoResult{Text: "Answered: YES!"}, nil
	}
	return EchoResult{Text: "Answered: NO!"}, nil
}

func main() {
	assets, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		panic(err)
	}

	app := &App{}

	// BindTyped — the CLI generates a type-safe bindings.d.ts from the lines below.
	hfd.BindTyped("Greet", app.Greet)
	hfd.BindTyped("SysInfo", app.SysInfo)
	hfd.BindTyped("StartTick", app.StartTick)
	hfd.BindTyped("PickFile", app.PickFile)
	hfd.BindTyped("SaveText", app.SaveText)
	hfd.BindTyped("ClipboardSet", app.ClipboardSet)
	hfd.BindTyped("ClipboardGet", app.ClipboardGet)
	hfd.BindTyped("SendNotification", app.SendNotification)
	hfd.BindTyped("AskConfirm", app.AskConfirm)

	// Tray (Phase 4): register after Run starts the engine — goroutine waits briefly.
	go func() {
		for i := 0; i < 50; i++ {
			err := hfd.SetTray(hfd.TrayOptions{
				Tooltip:  "HFD Demo — Go + GTK",
				IconPath: "applications-system", // theme icon (or a PNG file)
				Items: []hfd.TrayItem{
					{Label: "Fire a tick", Action: "tray-tick"},
					{Label: "About", Action: "tray-about"},
					{Label: "Quit", Action: "tray-quit"},
				},
			})
			if err == nil {
				log.Println("tray: registered OK")
				return
			}
			// not Run yet → retry; other error → give up.
			if !strings.Contains(err.Error(), "before Run") {
				log.Println("tray:", err)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err := hfd.Run(hfd.Options{
		Title:  "HFD Demo — Phase 3+4",
		Width:  880,
		Height: 700,
		Assets: assets,
		Debug:  true,
		CSP:    "default-src hfd://app/; script-src hfd://app/; style-src hfd://app/; img-src hfd://app/ data:; connect-src ws://localhost:5173 http://localhost:5173",
		// GTK menu bar — item click → JS event hfd.on("menu") {action}
		MenuLabel: "HFD",
		Menu: []hfd.MenuItem{
			{Label: "Open file…", Action: "open-file"},
			{Label: "Send notification", Action: "notify"},
			{Label: "About", Action: "about"},
		},
	}); err != nil {
		log.Println("hfd:", err)
		os.Exit(1)
	}
}
