// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/options"
	"v.io/x/ref/services/mounttable/mounttablelib"
	"v.io/x/ref/test"
	"v.io/x/ref/test/modules"
)

//go:generate v23 test generate

var rootMT = modules.Register(func(env *modules.Env, args ...string) error {
	ctx, shutdown := v23.Init()
	defer shutdown()

	lspec := v23.GetListenSpec(ctx)
	server, err := v23.NewServer(ctx, options.ServesMountTable(true))
	if err != nil {
		return fmt.Errorf("root failed: %v", err)
	}
	mt, err := mounttablelib.NewMountTableDispatcher("", "", "mounttable")
	if err != nil {
		return fmt.Errorf("mounttable.NewMountTableDispatcher failed: %s", err)
	}
	eps, err := server.Listen(lspec)
	if err != nil {
		return fmt.Errorf("server.Listen failed: %s", err)
	}
	if err := server.ServeDispatcher("", mt); err != nil {
		return fmt.Errorf("root failed: %s", err)
	}
	fmt.Fprintf(env.Stdout, "PID=%d\n", os.Getpid())
	for _, ep := range eps {
		fmt.Fprintf(env.Stdout, "MT_NAME=%s\n", ep.Name())
	}
	modules.WaitForEOF(env.Stdin)
	return nil
}, "rootMT")

// Starts a mounttable.  Returns the name and a stop function.
func startMountTable(t *testing.T, ctx *context.T) (string, func()) {
	sh, err := modules.NewShell(ctx, nil, testing.Verbose(), t)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	mt, err := sh.Start(nil, rootMT, "--v23.tcp.address=127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start root mount table: %s", err)
	}
	mt.ExpectVar("PID")
	rootName := mt.ExpectVar("MT_NAME")

	return rootName, func() {
		if err := sh.Cleanup(nil, nil); err != nil {
			t.Fatalf("failed to cleanup shell: %s", mt.Error())
		}
	}
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
	ctx, shutdown := test.V23Init()
	defer shutdown()

	mounttable, stopMountTable := startMountTable(t, ctx)
	defer stopMountTable()

	proxy := ""

	path := "path/to/channel"

	// Create a new channel.
	channel, err := newChannel(ctx, mounttable, proxy, path)
	if err != nil {
		t.Fatalf("newChannel(%v, %v, %v, %v) failed: %v", ctx, mounttable, proxy, path, err)
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
	channel2, err := newChannel(ctx, mounttable, proxy, path)
	if err != nil {
		t.Fatalf("newChannel(%v, %v, %v, %v) failed: %v", ctx, mounttable, proxy, path, err)
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

func TestBroadcastMessage(t *testing.T) {
	ctx, shutdown := test.V23Init()
	defer shutdown()

	mounttable, stopMountTable := startMountTable(t, ctx)
	defer stopMountTable()

	proxy := ""

	path := "path/to/channel"

	channel, err := newChannel(ctx, mounttable, proxy, path)
	if err != nil {
		t.Fatalf("newChannel(%v, %v, %v, %v) failed: %v", ctx, mounttable, proxy, path, err)
	}

	defer channel.leave()

	if err := channel.join(); err != nil {
		t.Fatalf("channel.join() failed: %v", err)
	}

	message := "Hello Vanadium world!"

	go func() {
		// Call getMembers(), which will set channel.members, used by
		// channel.broadcastMessage().
		if _, err := channel.getMembers(); err != nil {
			t.Fatalf("channel.getMembers() failed: %v", err)
		}
		if err := channel.broadcastMessage(message); err != nil {
			t.Fatalf("channel.broadcastMessage(%v) failed: %v", message, err)
		}
	}()

	select {
	case <-time.After(10 * time.Second):
		t.Errorf("Timeout waiting for message to be received.")
	case m := <-channel.messages:
		if m.Text != message {
			t.Errorf("Expected message text to be %v but got %v", message, m.Text)
		}
		if got, want := m.SenderName, channel.UserName(); got != want {
			t.Errorf("Got m.SenderName = %v, want %v", got, want)
		}
	}
}
