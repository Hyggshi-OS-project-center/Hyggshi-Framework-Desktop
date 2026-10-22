package hfd

import "errors"

// SetTitle đổi tiêu đề của cửa sổ lúc chạy.
func SetTitle(title string) error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: SetTitle trước khi Run")
	}
	return b.setTitle(title)
}

// CloseWindow đóng cửa sổ (kết thúc main loop → Run trả về).
func CloseWindow() error {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return errors.New("hfd: CloseWindow trước khi Run")
	}
	return b.closeWindow()
}
