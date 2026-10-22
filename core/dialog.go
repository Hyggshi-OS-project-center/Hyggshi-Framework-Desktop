package hfd

// Native dialog file — chạy modal trên GTK (chặn thư مربột UI cho tới khi
// chọn hay hủy). Gọi được từ handler goroutine (IPC) hoặc goroutine riêng.

import "errors"

// FileFilter một bộ lọc cho dialogs: Name hiển thị, Patterns vd ["*.png", "*.jpg"].
type FileFilter struct {
	Name     string
	Patterns []string
}

// FileDialogOptions cấu hình OpenFileDialog / SaveFileDialog.
type FileDialogOptions struct {
	Title       string
	Filters     []FileFilter
	StartDir    string // OpenFileDialog: thư mục mở sẵn
	InitialName string // SaveFileDialog: tên file gợi ý
}

// OpenFileDialog mở dialog chọn file. Trả về path và ok (false = user hủy).
func OpenFileDialog(opts FileDialogOptions) (path string, ok bool, err error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return "", false, errors.New("hfd: OpenFileDialog trước khi Run")
	}
	return b.openFileDialog(opts.Title, serializeFilters(opts.Filters), opts.StartDir)
}

// SaveFileDialog mở dialog lưu file (có xác nhận ghi đè).
func SaveFileDialog(opts FileDialogOptions) (path string, ok bool, err error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return "", false, errors.New("hfd: SaveFileDialog trước khi Run")
	}
	return b.saveFileDialog(opts.Title, opts.InitialName)
}

// MessageLevel mức của ShowMessage.
type MessageLevel int

const (
	MessageInfo MessageLevel = iota // nút OK
	MessageWarning                  // nút OK
	MessageError                    // nút OK
	MessageQuestion                 // nút Yes/No
)

// MessageDialogOptions cấu hình ShowMessage.
type MessageDialogOptions struct {
	Title   string
	Message string
	Level   MessageLevel
}

// ShowMessage hiện modal thông báo. Trả về true khi nút OK/Yes được bấm
// (hay false khi No/hủy — chỉ có nghĩa với MessageQuestion).
func ShowMessage(opts MessageDialogOptions) (ok bool, err error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return false, errors.New("hfd: ShowMessage trước khi Run")
	}
	return b.showMessage(opts.Title, opts.Message, int(opts.Level))
}

// serializeFilters chuyển filters về format C helper đọc:
//
//	Tên\tpattern1,pattern2\n...
//
// (kí tự kẻo không chứa \n\t,)
func serializeFilters(filters []FileFilter) string {
	s := ""
	for _, f := range filters {
		if len(f.Patterns) == 0 {
			continue
		}
		name := sanitizeLabel(f.Name, "Filter")
		pats := ""
		for i, p := range f.Patterns {
			if i > 0 {
				pats += ","
			}
			pats += sanitizeLabel(p, "*")
		}
		s += name + "\t" + pats + "\n"
	}
	return s
}

func sanitizeLabel(s, fallback string) string {
	// bỏ \n \t và , để không phá serialization
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\n' || c == '\t' || c == ',' || c < 0x20 {
			continue
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return fallback
	}
	return string(out)
}
