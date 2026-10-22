package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"runtime"

	"github.com/hyggshi/hfd"
)

//go:embed all:frontend/dist
var distFS embed.FS

type GreetArgs struct {
	Name string `json:"name"`
}

type GreetResult struct {
	Message string `json:"message"`
}

type SysInfoArgs struct{}

type SysInfoResult struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	CPUs     int    `json:"cpus"`
	GoVer    string `json:"goVersion"`
}

type App struct{}

func (a *App) Greet(args GreetArgs) (GreetResult, error) {
	if args.Name == "" {
		args.Name = "bạn"
	}
	return GreetResult{Message: "Xin chào " + args.Name + " từ HFD!"}, nil
}

func (a *App) SysInfo(_ SysInfoArgs) (SysInfoResult, error) {
	hostname, _ := os.Hostname()
	return SysInfoResult{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUs:     runtime.NumCPU(),
		GoVer:    runtime.Version(),
	}, nil
}

func main() {
	assets, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		panic(err)
	}

	app := &App{}
	hfd.BindTyped("Greet", app.Greet)
	hfd.BindTyped("SysInfo", app.SysInfo)

	if err := hfd.Run(hfd.Options{
		Title:  "react-demo",
		Width:  900,
		Height: 640,
		Assets: assets,
		Debug:  true,
	}); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
