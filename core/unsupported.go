//go:build !linux && !windows && !darwin

package hfd

import "errors"

const notLinuxMsg = "hfd: backend Linux (GTK + WebKitGTK) mới — nền tảng này ở Phase 5"

type stubBackend struct{}

func newEngine(opts Options) backend { return stubBackend{} }

func (stubBackend) run() error                               { return errors.New(notLinuxMsg) }
func (stubBackend) emit(name string, payload any) error       { return errors.New(notLinuxMsg) }
func (stubBackend) setClipboard(string) error                 { return errors.New(notLinuxMsg) }
func (stubBackend) getClipboard() (string, error)             { return "", errors.New(notLinuxMsg) }
func (stubBackend) openFileDialog(string, string, string) (string, bool, error) {
	return "", false, errors.New(notLinuxMsg)
}
func (stubBackend) saveFileDialog(string, string) (string, bool, error) {
	return "", false, errors.New(notLinuxMsg)
}
func (stubBackend) showMessage(string, string, int) (bool, error) { return false, errors.New(notLinuxMsg) }
func (stubBackend) setTitle(string) error                        { return errors.New(notLinuxMsg) }
func (stubBackend) closeWindow() error                          { return errors.New(notLinuxMsg) }
