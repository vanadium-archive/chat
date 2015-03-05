package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/nlacasse/gocui"

	"v.io/v23"
	"v.io/v23/naming"

	"v.io/x/lib/vlog"
)

var (
	mounttable = flag.String("mounttable", "/proxy.envyor.com:8101", "Mounttable where channel is mounted.")
	// TODO(nlacasse): Allow this to be set by a flag.
	appName     = "apps/chat"
	channelName = "public"
)

const welcomeText = `***Welcome to Vanadium Chat***
Press Ctrl-C to exit.
`

func init() {
	logDir, err := ioutil.TempDir("", "chat-logs")
	if err != nil {
		panic(err)
	}
	err = vlog.Log.Configure(vlog.LogDir(logDir))
	if err != nil {
		panic(err)
	}

	// Make sure that *nothing* ever gets printed to stderr.
	os.Stderr.Close()
}

// Defines the layout of the UI.
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	membersViewWidth := 30
	messageInputViewHeight := 3

	if _, err := g.SetView("history", -1, -1, maxX-membersViewWidth, maxY-messageInputViewHeight); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	if membersView, err := g.SetView("members", maxX-membersViewWidth, -1, maxX, maxY-messageInputViewHeight); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		membersView.FgColor = gocui.ColorCyan
	}
	if messageInputView, err := g.SetView("messageInput", -1, maxY-messageInputViewHeight, maxX, maxY-1); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		messageInputView.Editable = true
	}
	if err := g.SetCurrentView("messageInput"); err != nil {
		return err
	}
	return nil
}

// app encapsulates the UI and the channel logic.
type app struct {
	cr            *channel
	g             *gocui.Gui
	hw            *historyWriter
	cachedMembers []string
	// Function to call when shutting down the app.
	shutdown func()
	// Mutex to protect read/writes to cachedMembers array.
	mu sync.Mutex
}

// Initialize the UI and channel.
func newApp() *app {
	// Set up the UI.
	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		log.Panicln(err)
	}
	g.ShowCursor = true
	g.SetLayout(layout)

	// Draw the layout.
	g.Flush()

	ctx, ctxShutdown := v23.Init()

	shutdown := func() {
		ctxShutdown()
		g.Close()
	}

	cr, err := newChannel(ctx, *mounttable, naming.Join(appName, channelName))
	if err != nil {
		log.Panicln(err)
	}

	hw := newHistoryWriter(g.View("history"), cr.UserName())
	hw.Write([]byte(color.RedString(welcomeText)))

	hw.Write([]byte(fmt.Sprintf("You have joined channel '%s' on mounttable '%s'.\n"+
		"Your username is '%s'.\n\n", channelName, *mounttable, cr.UserName())))

	a := &app{
		cr:       cr,
		g:        g,
		hw:       hw,
		shutdown: shutdown,
	}

	if err := a.setKeybindings(); err != nil {
		log.Panicln(err)
	}

	return a
}

// Helper method to log to the history console when debugging.
func (a *app) log(m string) {
	a.hw.Write([]byte("LOG: " + m + "\n"))
}

func (a *app) quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrorQuit
}

func (a *app) handleSendMessage(g *gocui.Gui, v *gocui.View) error {
	text := strings.TrimSpace(v.Buffer())
	if text == "" {
		return nil
	}
	if err := a.cr.broadcastMessage(text); err != nil {
		return err
	}
	v.Clear()
	return nil
}

func (a *app) handleTabComplete(g *gocui.Gui, v *gocui.View) error {
	lastWord, err := v.Word(v.Cursor())
	if err != nil {
		// The view buffer is empty.  Just return early.
		return nil
	}

	// Get a list of names that match the last word.
	matchedNames := []string{}
	a.mu.Lock()
	for _, name := range a.cachedMembers {
		if strings.HasPrefix(name, lastWord) {
			matchedNames = append(matchedNames, name)
		}
	}
	a.mu.Unlock()

	if len(matchedNames) == 0 {
		return nil
	}

	// Get the longest common prefix between the matchedNames.
	lcp := longestCommonPrefix(matchedNames)

	// Suffix is the part of the lcp that is not already part of the
	// lastWord.
	suffix := lcp[len(lastWord):]

	// If the name was matched uniquely, append a space.
	if len(matchedNames) == 1 {
		suffix = suffix + " "
	}

	// Simply writing the suffix to the buffer causes strange whitespace
	// additions. To work around this, we calculate the desired content of
	// the buffer, then clear the buffer and write the entire new content.
	newLine := strings.TrimSpace(v.Buffer()) + suffix
	v.Clear()
	v.Write([]byte(newLine))

	// Set the cursor to the end of the new line, and reset the origin.
	v.SetCursor(len(newLine), 0)
	v.SetOrigin(0, 0)

	return nil
}

func (a *app) setKeybindings() error {
	// Ctrl-C => Exit.
	if err := a.g.SetKeybinding("", gocui.KeyCtrlC, 0, a.quit); err != nil {
		return err
	}

	// Enter => Send message.
	if err := a.g.SetKeybinding("messageInput", gocui.KeyEnter, 0, a.handleSendMessage); err != nil {
		return err
	}

	// Tab => Tab-complete member names.
	if err := a.g.SetKeybinding("messageInput", gocui.KeyTab, 0, a.handleTabComplete); err != nil {
		return err
	}

	return nil
}

// updateMembers gets the members from the channel and writes them to the
// members view. It also caches the members in app.cachedMembers for use in tab
// autocomplete.
func (a *app) updateMembers() {
	membersView := a.g.View("members")

	members, err := a.cr.getMembers()
	if err != nil {
		log.Panicln(err)
	}

	newCachedMembers := []string{}
	membersView.Clear()

	for _, member := range members {
		newCachedMembers = append(newCachedMembers, member.Name)
		membersView.Write([]byte(member.Name + "\n"))
	}

	a.mu.Lock()
	a.cachedMembers = newCachedMembers
	a.mu.Unlock()
	a.g.Flush()
}

// displayIncomingMessages listens for incoming messages and writes them to the
// historyWriter.
func (a *app) displayIncomingMessages() {
	go func() {
		for {
			m := <-a.cr.messages
			a.hw.writeMessage(m)
		}
	}()
}

// run joins the channel and starts the main app loop.
func (a *app) run() error {
	// Join the channel.
	if err := a.cr.join(); err != nil {
		log.Panicln(err)
	}
	defer a.cr.leave()

	// Update the members view in a loop.
	go func() {
		for {
			a.updateMembers()
			time.Sleep(2 * time.Second)
		}
	}()

	a.displayIncomingMessages()

	// Start the main UI loop.
	if err := a.g.MainLoop(); err != nil && err != gocui.ErrorQuit {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	a := newApp()
	defer a.shutdown()
	if err := a.run(); err != nil {
		log.Panicln(err)
	}
}
