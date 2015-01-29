package main

import (
	"fmt"
	"io"
	"regexp"
	"sync"

	"github.com/fatih/color"
	"github.com/kr/text"
	"github.com/nlacasse/gocui"
)

var yellow = color.New(color.FgYellow).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()

// historyWriter wraps the history view.  All text written to the history view
// UI component is written though a history writer, which has methods to word
// wrap text to the view, highlight the users name, and format messages.  When
// messages are received, they are sent to the view through the "writeMessage"
// method.
type historyWriter struct {
	// Mutex to prevent concurrent  writes to the view buffer.
	mu             sync.Mutex
	userName       string
	userNameRegexp *regexp.Regexp
	view           *gocui.View
}

var _ io.Writer = (*historyWriter)(nil)

// newHistoryWriter creates a new historyWriter for the given view and
// username.  The username will be highlighted in message text.
func newHistoryWriter(view *gocui.View, userName string) *historyWriter {
	return &historyWriter{
		userName:       userName,
		userNameRegexp: regexp.MustCompile("(?i)" + userName),
		view:           view,
	}
}

// Write wraps the view.Write method.  It is exported so that historyWriter
// satisfies the Writer interface.
func (hw *historyWriter) Write(b []byte) (int, error) {
	return hw.view.Write(b)
}

// writeWordWrap wraps the text to the width of the view and writes it to the
// buffer.  It also scrolls the text up if the buffer is longer than the height
// of the view.
func (hw *historyWriter) writeWordWrap(b []byte) {
	width, height := hw.view.Size()
	hw.mu.Lock()
	defer hw.mu.Unlock()
	hw.Write(text.WrapBytes(b, width))
	numLines := hw.view.NumberOfLines()
	if numLines > height {
		hw.view.SetOrigin(0, numLines-height)
	}
}

func (hw *historyWriter) highlightUserName(st string) string {
	return hw.userNameRegexp.ReplaceAllLiteralString(st, cyan(hw.userName))
}

// TODO(nlacasse): Consider coloring each sender name with a unique color.
func (hw *historyWriter) formatMessage(m message) string {
	const timeFormat = "Jan 2 at 3:04pm"
	t := m.Timestamp.Format(timeFormat)

	return fmt.Sprintf("%s %s: %s\n", yellow(t), cyan(m.Sender.ShortName()), hw.highlightUserName(m.Text))
}

// writeMessage formats a message and writes.
func (hw *historyWriter) writeMessage(m message) {
	f := hw.formatMessage(m)
	hw.writeWordWrap([]byte(f))
}
