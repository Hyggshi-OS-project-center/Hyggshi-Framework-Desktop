package hfd

import (
	"errors"
	"os/exec"
)

// Notify gửi desktop notification (qua notify-send trên linux — Quickstart;
// sẽ sang GNotification/D-Bus trực tiếp ở phase sau).
func Notify(title, body string) error {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		return errors.New("hfd: Notify cần libnotify-bin (notify-send) hoặc dùng phase sau")
	}
	if body == "" {
		return exec.Command(path, title).Run()
	}
	return exec.Command(path, "--app-name=HFD", title, body).Run()
}
