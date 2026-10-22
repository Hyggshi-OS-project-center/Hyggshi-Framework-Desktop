package hfd

// Native dialog file — shows a modal over GTK (blocking the UI event loop
// until selection or cancel). Callable from a handler goroutine (IPC) or a
// separate goroutine.

import "errors"

// FileFilter is a filter for dialogs: Name is displayed, Patterns e.g. ["*.png", "*.jpg"].
type FileFilter struct {
	Name     string
	Patterns []string
}

// FileDialogOptions configures OpenFileDialog / SaveFileDialog.
type FileDialogOptions struct {
	Title       string
	Filters     []FileFilter
	StartDir    string // OpenFileDialog: directory opened initially
	InitialName string // SaveFileDialog: suggested file name
}

// OpenFileDialog opens a file-picker dialog. Returns path and ok (false = user canceled).
func OpenFileDialog(opts FileDialogOptions) (path string, ok bool, err error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return "", false, errors.New("hfd: OpenFileDialog before Run")
	}
	return b.openFileDialog(opts.Title, serializeFilters(opts.Filters), opts.StartDir)
}

// SaveFileDialog opens a save-file dialog (with overwrite confirmation).
func SaveFileDialog(opts FileDialogOptions) (path string, ok bool, err error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return "", false, errors.New("hfd: SaveFileDialog before Run")
	}
	return b.saveFileDialog(opts.Title, opts.InitialName)
}

// MessageLevel is the level of ShowMessage.
type MessageLevel int

const (
	MessageInfo MessageLevel = iota // OK button
	MessageWarning                  // OK button
	MessageError                    // OK button
	MessageQuestion                 // Yes/No buttons
)

// MessageDialogOptions configures ShowMessage.
type MessageDialogOptions struct {
	Title   string
	Message string
	Level   MessageLevel
}

// ShowMessage shows a message modal. Returns true when the OK/Yes button is
// clicked (or false on No/cancel — only meaningful for MessageQuestion).
func ShowMessage(opts MessageDialogOptions) (ok bool, err error) {
	curMu.Lock()
	b := curBackend
	curMu.Unlock()
	if b == nil {
		return false, errors.New("hfd: ShowMessage before Run")
	}
	return b.showMessage(opts.Title, opts.Message, int(opts.Level))
}

// serializeFilters converts the filters into the format the C helper reads:
//
//	Name\tpattern1,pattern2\n...
//
// (characters stripped so the values cannot contain \n\t,)
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
	// strip \n, \t and , so the serialization isn't broken
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
