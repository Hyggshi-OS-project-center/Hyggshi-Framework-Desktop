package hfd

import (
	"errors"
	"os/exec"
)

// Notify sends a desktop notification (via notify-send on linux — Quickstart;
// switches to GNotification/D-Bus directly in a later phase).
func Notify(title, body string) error {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		return errors.New("hfd: Notify requires libnotify-bin (notify-send), or a later phase")
	}
	if body == "" {
		return exec.Command(path, title).Run()
	}
	return exec.Command(path, "--app-name=HFD", title, body).Run()
}
