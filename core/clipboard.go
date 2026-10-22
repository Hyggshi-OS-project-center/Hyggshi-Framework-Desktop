package hfd

import "errors"

// SetClipboard writes text to the system clipboard.
func SetClipboard(text string) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: SetClipboard before Run")
	}
	return b.setClipboard(text)
}

// GetClipboard reads text from the system clipboard (empty if there is no text).
func GetClipboard() (string, error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return "", errors.New("hfd: GetClipboard before Run")
	}
	return b.getClipboard()
}
