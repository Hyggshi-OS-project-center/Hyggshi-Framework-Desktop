package hfd

import "errors"

// SetClipboard ghi văn bản vào clipboard hệ thống.
func SetClipboard(text string) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: SetClipboard trước khi Run")
	}
	return b.setClipboard(text)
}

// GetClipboard đọc văn bản từ clipboard hệ thống (rỗng nếu không có text).
func GetClipboard() (string, error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return "", errors.New("hfd: GetClipboard trước khi Run")
	}
	return b.getClipboard()
}
