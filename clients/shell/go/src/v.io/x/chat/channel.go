// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

// channel holds logic for finding and communicating with members of a
// channel.
//
// Usage:
//  // Construct a new channel.
//  c := newChannel(ctx, mounttable, proxy, "path/to/channel/name")
//
//  // Join the channel.
//  err := c.join()
//
//  // Get all members in the channel.
//  members, err := c.getMembers()
//
//  // Send a message to a member.
//  c.sendMessageTo(member, "message")
//
//  // Send a message to all members in the channel.
//  c.broadcastMessage("message")
//
//  // Leave the channel.
//  c.leave()

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/naming"
	"v.io/v23/options"
	"v.io/v23/rpc"
	"v.io/v23/security"
	"v.io/v23/security/access"
	mt "v.io/v23/services/mounttable"
	"v.io/x/chat/vdl"
	_ "v.io/x/ref/runtime/factories/roaming"
)

// message is a message that will be displayed in the UI.
type message struct {
	SenderName string
	Text       string
	Timestamp  time.Time
}

// chatServerMethods implements the chat server VDL interface.
type chatServerMethods struct {
	// Incoming messages get sent to messages channel.
	messages chan<- message
}

var _ vdl.ChatServerMethods = (*chatServerMethods)(nil)

func newChatServerMethods(messages chan<- message) *chatServerMethods {
	return &chatServerMethods{
		messages: messages,
	}
}

// SendMessage is called by clients to send a message to the server.
func (cs *chatServerMethods) SendMessage(ctx *context.T, call rpc.ServerCall, IncomingMessage string) error {
	remoteb, _ := security.RemoteBlessingNames(ctx, call.Security())
	cs.messages <- message{
		SenderName: firstShortName(remoteb),
		Text:       IncomingMessage,
		Timestamp:  time.Now(),
	}
	return nil
}

// member is a member of the channel.
type member struct {
	// Blessings is the remote blessings of the member.  There could
	// potentially be multiple.
	Blessings []string
	// Name is the name we will display for this member.
	Name string
	// Path is the path in the mounttable where the member is mounted.
	Path string
}

// members are sortable by Name.
type byName []*member

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }

// channel interface.
type channel struct {
	// Vanadium context.
	ctx *context.T
	// The location where we mount ourselves and look for other users.
	path string
	// The implementation of the chat server.
	chatServerMethods *chatServerMethods
	// The chat server.
	server rpc.Server
	// Channel that emits incoming messages.
	messages chan message
	// Cached list of channel members.
	members []*member
}

func newChannel(ctx *context.T, mounttable, proxy, path string) (*channel, error) {
	// Set the namespace root to the mounttable passed on the command line.
	newCtx, _, err := v23.WithNewNamespace(ctx, mounttable)
	if err != nil {
		return nil, err
	}

	// Set the proxy that will be used to listen.
	listenSpec := v23.GetListenSpec(ctx)
	listenSpec.Proxy = proxy

	messages := make(chan message)

	return &channel{
		chatServerMethods: newChatServerMethods(messages),
		messages:          messages,
		path:              path,
		ctx:               newCtx,
		server:            nil,
	}, nil
}

// UserName returns a short, human-friendly representation of the chat client.
func (cr *channel) UserName() string {
	// TODO(ashankar): It is wrong to assume that
	// v23.GetPrincipal(ctx).BlessingStore().Default() returns a valid
	// "sender". Think about the "who-am-I" API and use that here instead.
	userName := fmt.Sprint(v23.GetPrincipal(cr.ctx).BlessingStore().Default())
	if sn := shortName(userName); sn != "" {
		userName = sn
	}
	return userName
}

// getLockedName picks a random name inside the channel's mounttable path and
// tries to "lock" it by settings restrictive permissions on the name.  It
// tries repeatedly until it finds an unused name that can be locked, and
// returns the locked name.
func (cr *channel) getLockedName() (string, error) {
	myPatterns := security.DefaultBlessingPatterns(v23.GetPrincipal(cr.ctx))

	// myACL is an ACL that only allows my blessing.
	myACL := access.AccessList{
		In: myPatterns,
	}
	// openACL is an ACL that allows anybody.
	openACL := access.AccessList{
		In: []security.BlessingPattern{security.AllPrincipals},
	}

	permissions := access.Permissions{
		// Give everybody the ability to read and resolve the name.
		string(mt.Resolve): openACL,
		string(mt.Read):    openACL,
		// All other permissions are only for us.
		string(mt.Admin):  myACL,
		string(mt.Create): myACL,
		string(mt.Mount):  myACL,
	}

	// Repeatedly try to SetPermissions under random names until we find a free
	// one.

	// Collisions should be rare.  25 times should be enough to find a free
	// one
	maxTries := 25
	for i := 0; i < maxTries; i++ {
		// Pick a random suffix, the hash of our default blessing and the time.
		now := time.Now().UnixNano()
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", cr.UserName(), now)))
		suffix := base64.URLEncoding.EncodeToString(hash[:])

		name := naming.Join(cr.path, suffix)

		ns := v23.GetNamespace(cr.ctx)

		if err := ns.SetPermissions(cr.ctx, name, permissions, ""); err != nil {
			// Try again with a different name.
			continue
		}

		// SetPermissions succeeded!  We now own the name.
		return name, nil
	}
	return "", fmt.Errorf("Error getting a locked name.  Tried %v times but did not succeed.", maxTries)
}

// join starts a chat server and mounts it in the channel path.
func (cr *channel) join() error {
	// Get a locked name in the mounttable that we can mount our server on.
	name, err := cr.getLockedName()
	if err != nil {
		return err
	}
	// Serve the chat server on the locked name.
	serverChat := vdl.ChatServer(cr.chatServerMethods)

	// Create a new server.
	_, cr.server, err = v23.WithNewServer(cr.ctx, name, serverChat, security.AllowEveryone())
	return err
}

// leave stops the chat server and removes our mounted name from the
// mounttable.
func (cr *channel) leave() error {
	// Stop serving.
	cr.server.Stop()

	// Get the names we are mounted at.  Should only be one.
	names := cr.server.Status().Mounts.Names()
	// Delete the name and all sub-names in the hierarchy.
	ns := v23.GetNamespace(cr.ctx)
	for _, name := range names {
		if err := ns.Delete(cr.ctx, name, true); err != nil {
			return err
		}
	}

	cr.server = nil

	return nil
}

// newMember creates a new member object.
func (cr *channel) newMember(blessings []string, path string) *member {
	name := "unknown"
	if len(blessings) > 0 {
		// Arbitrarily choose the first blessing as the display name.
		name = shortName(blessings[0])
	}
	return &member{
		Name:      name,
		Blessings: blessings,
		Path:      path,
	}
}

// getMembers gets a list of members in the channel.
func (cr *channel) getMembers() ([]*member, error) {
	ctx, cancel := context.WithTimeout(cr.ctx, 5*time.Second)
	defer cancel()

	// Glob on the channel path for mounted members.
	globPath := cr.path + "/*"
	globChan, err := v23.GetNamespace(ctx).Glob(ctx, globPath)
	if err != nil {
		return nil, err
	}

	members := []*member{}

	for reply := range globChan {
		switch v := reply.(type) {
		case *naming.GlobReplyEntry:
			blessings := blessingNamesFromMountEntry(&v.Value)
			if len(blessings) == 0 {
				// No servers mounted at that name, likely only a
				// lonely ACL.  Safe to ignore.
				// TODO(nlacasse): Should there be a time-limit
				// on ACLs in the namespace?  Seems like we'll
				// have an ACL graveyard before too long.
				continue
			}
			member := cr.newMember(blessings, v.Value.Name)
			members = append(members, member)
		}
	}

	sort.Sort(byName(members))

	cr.members = members
	return members, nil
}

// broadcastMessage sends a message to all members in the channel.
func (cr *channel) broadcastMessage(messageText string) error {
	for _, member := range cr.members {
		// TODO(nlacasse): Sending messages async means they might get sent out of
		// order. Consider either sending them sync or maintain a queue.
		go cr.sendMessageTo(member, messageText)
	}
	return nil
}

// sendMessageTo sends a message to a particular member.  It ensures that the
// receiving server has the same blessings that the member does.
func (cr *channel) sendMessageTo(member *member, messageText string) {
	ctx, cancel := context.WithTimeout(cr.ctx, 5*time.Second)
	defer cancel()

	s := vdl.ChatClient(member.Path)

	var opts []rpc.CallOpt
	if len(member.Blessings) > 0 {
		// The server must match the blessings we got when we globbed it.
		// The AllowedServersPolicy options require that the server matches the
		acl := access.AccessList{In: make([]security.BlessingPattern, len(member.Blessings))}
		for i, b := range member.Blessings {
			acl.In[i] = security.BlessingPattern(b)
		}
		opts = append(opts, options.ServerAuthorizer{acl})
	}
	if err := s.SendMessage(ctx, messageText, opts...); err != nil {
		return // member has disconnected.
	}
}

func blessingNamesFromMountEntry(me *naming.MountEntry) []string {
	names := me.Names()
	if len(names) == 0 {
		return nil
	}
	// Using the first valid mount entry for now.
	// TODO(nlacasse): How should we deal with multiple members mounted on
	// a single mountpoint?
	for _, name := range names {
		addr, _ := naming.SplitAddressName(name)
		ep, err := v23.NewEndpoint(addr)
		if err != nil {
			// TODO(nlacasse): Log this or bubble up?
			continue
		}
		return ep.BlessingNames()
	}
	return nil
}
