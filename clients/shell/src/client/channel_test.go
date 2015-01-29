package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"testing"
	"time"

	"v.io/core/veyron2"
	"v.io/core/veyron2/context"
)

// Creates a new context and a new mounttable. Returns the context, mounttable
// endpoint, and a teardown function.
func setup(t *testing.T) (*context.T, string, func(*testing.T)) {
	ctx, shutdown := veyron2.Init()

	mtProc, endpoint, err := startMounttabled()
	if err != nil {
		t.Fatal(err)
	}

	teardown := func(t *testing.T) {
		shutdown()
		if err := mtProc.Kill(); err != nil {
			t.Fatalf("Error killing mounttabled: %v", err)
		}
	}
	return ctx, endpoint, teardown
}

// Asserts that the channel contains members with expected names and no others.
func AssertMembersWithNames(channel *channel, expectedNames []string) error {
	members, err := channel.getMembers()
	if err != nil {
		return fmt.Errorf("channel.getMembers() failed: %v", err)
	}
	if actualLen, expectedLen := len(members), len(expectedNames); actualLen != expectedLen {
		return fmt.Errorf("Wrong number of members.  Expected %v, actual %v.", expectedLen, actualLen)
	}
	for _, expectedName := range expectedNames {
		found := false
		for _, member := range members {
			if member.Name == expectedName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Expected member with name %v, but did not find one.", expectedName)
		}
	}
	return nil
}

func TestMembers(t *testing.T) {
	ctx, mounttable, teardown := setup(t)
	defer teardown(t)

	path := "path/to/channel"

	// Create a new channel.
	channel, err := newChannel(ctx, mounttable, path)
	if err != nil {
		t.Fatal("newChannel(%v, %v, %v) failed: %v", ctx, mounttable, path)
	}

	// New channel should be empty.
	if err := AssertMembersWithNames(channel, []string{}); err != nil {
		t.Error(err)
	}

	// Join the channel.
	if err := channel.join(); err != nil {
		t.Fatalf("channel.join() failed: %v", err)
	}

	// Channel should contain only current user.
	if err := AssertMembersWithNames(channel, []string{channel.UserName()}); err != nil {
		t.Error(err)
	}

	// Create and join the channel a second time.
	channel2, err := newChannel(ctx, mounttable, path)
	if err != nil {
		t.Fatal("newChannel(%v, %v, %v) failed: %v", ctx, mounttable, path)
	}
	if err := channel2.join(); err != nil {
		t.Fatalf("channel2.join() failed: %v", err)
	}

	// Channel should contain both users.
	if err := AssertMembersWithNames(channel, []string{channel.UserName(), channel2.UserName()}); err != nil {
		t.Error(err)
	}

	// Leave first instance of channel.
	if err := channel.leave(); err != nil {
		t.Fatalf("channel.leave() failed: %v", err)
	}

	// Channel should contain only second user.
	if err := AssertMembersWithNames(channel, []string{channel2.UserName()}); err != nil {
		t.Error(err)
	}

	// Leave second instance of channel.
	if err := channel2.leave(); err != nil {
		t.Fatalf("channel2.leave() failed: %v", err)
	}

	// Channel should be empty.
	if err := AssertMembersWithNames(channel, []string{}); err != nil {
		t.Error(err)
	}
}

func testBroadcastMessage(t *testing.T) {
	ctx, mounttable, teardown := setup(t)
	defer teardown(t)

	path := "path/to/channel"

	channel, err := newChannel(ctx, mounttable, path)
	if err != nil {
		t.Fatalf("newChannel(%v, %v, %v) failed: %v", ctx, mounttable, path, err)
	}

	defer channel.leave()

	if err := channel.join(); err != nil {
		t.Fatalf("channel.join() failed: %v", err)
	}

	message := "Hello Vanadium world!"

	go func() {
		if err := channel.broadcastMessage(message); err != nil {
			t.Fatalf("channel.broadcastMessage(%v) failed: %v", message, err)
		}
	}()

	select {
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for message to be received.")
	case m := <-channel.messages:
		if m.Text != message {
			t.Errorf("Expected message text to be %v but got %v", message, m.Text)
		}
		if len(m.Sender) == 0 {
			t.Errorf("Expected message.Sender to be [%v] but got an empty set", channel.UserName())
		} else if got, want := m.Sender[0], channel.UserName(); got != want {
			t.Errorf("Got m.Sender[0] = %v, want %v", got, want)
		}
	}
}

// startMounttabled starts a mounttabled process on a random port, and returns
// its process and endpoint, or an error.
// TODO(nlacasse): We have similar logic in go and js tests and also in the
// playground for starting services and grepping for endpoints, ports, or other
// status text.  Consider making a go library that knows how to start common
// services and return relavent bits of information.
func startMounttabled() (*os.Process, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("os.Getwd() failed: %v", err)
	}

	mounttabled := path.Join(cwd, "..", "..", "bin", "mounttabled")

	cmd := exec.Command(mounttabled, "--veyron.tcp.address=127.0.0.1:0")
	timeLimit := 5 * time.Second
	matches, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile("Mount table .+ endpoint: (.+)\n"))
	if err != nil {
		return nil, "", fmt.Errorf("Error starting mounttabled: %v", err)
	}
	endpoint := matches[1]
	return cmd.Process, endpoint, nil
}

// Begin copy-paste from playground/builder/services.go.

// Helper function to start a command and wait for output.  Arguments are a cmd
// to run, a timeout, and a regexp.  The slice of strings matched by the regexp
// is returned.
func startAndWaitFor(cmd *exec.Cmd, timeout time.Duration, outputRegexp *regexp.Regexp) ([]string, error) {
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = cmd.Stdout
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	buf := bufio.NewReader(reader)
	t := time.After(timeout)
	ch := make(chan []string)
	go (func() {
		for line, err := buf.ReadString('\n'); err == nil; line, err = buf.ReadString('\n') {
			if matches := outputRegexp.FindStringSubmatch(line); matches != nil {
				ch <- matches
			}
		}
		close(ch)
	})()
	select {
	case <-t:
		return nil, fmt.Errorf("Timeout starting service.")
	case matches := <-ch:
		return matches, nil
	}
}

// End copy-paste from playground/builder/services.go
