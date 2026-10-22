package hfd

import "errors"

// SetTitle changes the window title at runtime.
func SetTitle(title string) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: SetTitle before Run")
	}
	return b.setTitle(title)
}

// CloseWindow closes the window (ends the main loop → Run returns).
func CloseWindow() error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: CloseWindow before Run")
	}
	return b.closeWindow()
}
