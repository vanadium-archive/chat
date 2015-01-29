package main

// channel holds logic for finding and communicating with members of a
// channel.
//
// Usage:
//  // Construct a new channel.
//  cr := newChannel("path/to/channel/name")
//
//  // Join the channel.
//  cr.join()
//
//  // Get all members in the channel.
//  members, err := cr.getMembers()
//
//  // Send a message to a member.
//  cr.sendMessageTo(member, "message")
//
//  // Send a message to all members in the channel.
//  cr.broadcastMessage("message")
//
//  // Leave the channel.
//  cr.leave()

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	_ "v.io/core/veyron/profiles/roaming"
	"v.io/core/veyron2"
	"v.io/core/veyron2/context"
	"v.io/core/veyron2/ipc"
	"v.io/core/veyron2/naming"
	"v.io/core/veyron2/security"
	"v.io/core/veyron2/vlog"

	// TODO(sadovsky): This should be under "github.com/veyron/chat" or somesuch.
	"service"
)

// Sender represents the blessings of the sender of the message.
type Sender []string

func (s Sender) ShortName() string {
	return firstShortName(s)
}

type message struct {
	Sender    Sender
	Text      string
	Timestamp time.Time
}

// Chat server interface.
type chatServerMethods struct {
	// Incoming messages get sent to messages channel.
	messages chan<- message
}

var _ service.ChatServerMethods = (*chatServerMethods)(nil)

func newChatServerMethods(messages chan<- message) *chatServerMethods {
	return &chatServerMethods{
		messages: messages,
	}
}

// SendMessage is called by clients to send a message to the server.
func (cs *chatServerMethods) SendMessage(ctx ipc.ServerContext, IncomingMessage string) error {
	var sender Sender
	sender = Sender(ctx.RemoteBlessings().ForContext(ctx))
	cs.messages <- message{
		Sender:    sender,
		Text:      IncomingMessage,
		Timestamp: time.Now(),
	}
	return nil
}

// member is a member of the channel.
type member struct {
	Name string
	Path string
}

// members are sortable by Name.
// Consider using https://github.com/pmylund/sortutil
type byName []*member

func (b byName) Len() int           { return len(b) }
func (b byName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byName) Less(i, j int) bool { return b[i].Name < b[j].Name }

// channel interface.
type channel struct {
	chatServerMethods *chatServerMethods
	// The location where we mount ourselves and look for other users.
	path   string
	ctx    *context.T
	server ipc.Server
	// Random number to allow different instances of clients, running
	// with the same blessings (e.g., id-provider/foo@bar.com/laptop)
	// to mount themselves under different names.
	instance int
	// Channel that emits incoming messages.
	messages chan message
}

func newChannel(ctx *context.T, mounttable, path string) (*channel, error) {
	// Set the namespace root to the mounttable passed on the command line.
	veyron2.SetNewNamespace(ctx, mounttable)

	// Turn off logging to stderr.
	vlog.Log.ConfigureLogger(
		vlog.LogToStderr(false),
		vlog.AlsoLogToStderr(false))

	s, err := veyron2.NewServer(ctx)
	if err != nil {
		return nil, err
	}

	rand.Seed(time.Now().UnixNano())
	messages := make(chan message)

	return &channel{
		chatServerMethods: newChatServerMethods(messages),
		messages:          messages,
		path:              path,
		ctx:               ctx,
		server:            s,
		instance:          rand.Intn(9999),
	}, nil
}

// openAuthorizer allows RPCs from all clients.
// TODO(nlacasse): Write a more strict authorizer once we have a better
// understanding of ACLs and identities, and once javascript supports similar
// functionality.
type openAuthorizer struct{}

func (o openAuthorizer) Authorize(_ security.Context) error {
	return nil
}

// UserName returns a short, human-friendly representation of the chat client.
func (cr *channel) UserName() string {
	// TODO(ashankar): It is wrong to assume that
	// veyron2.GetPrincipal(ctx).BlessingStore().Default() returns a valid
	// "sender". Think about the "who-am-I" API and use that here instead.
	userName := fmt.Sprint(veyron2.GetPrincipal(cr.ctx).BlessingStore().Default())
	if sn := shortName(userName); sn != "" {
		userName = sn
	}
	return fmt.Sprintf("%s-%04d", userName, cr.instance)
}

// join starts a chat server and mounts it in the channel path.
func (cr *channel) join() error {
	if _, err := cr.server.Listen(veyron2.GetListenSpec(cr.ctx)); err != nil {
		return err
	}
	serverChat := service.ChatServer(cr.chatServerMethods)
	path := naming.Join(cr.path, cr.UserName())
	if err := cr.server.Serve(path, serverChat, openAuthorizer{}); err != nil {
		return err
	}

	return nil
}

func (cr *channel) leave() error {
	return cr.server.Stop()
}

func (cr *channel) newMember(path string) (*member, error) {
	// The last part of the path is the name.
	name := path[strings.LastIndex(path, "/")+1 : len(path)]

	member := member{
		Name: name,
		Path: path,
	}
	return &member, nil
}

// getMembers gets a list of members in the channel.
// TODO(nlacasse): Figure out how to get the Blessings (and hence the emails) of
// the members. Right now we just trust that they mounted under their actual
// email address.
func (cr *channel) getMembers() ([]*member, error) {
	ctx, cancel := context.WithTimeout(cr.ctx, 5*time.Second)
	defer cancel()

	globPath := cr.path + "/*"

	memberChan, err := veyron2.GetNamespace(ctx).Glob(ctx, globPath)
	if err != nil {
		return nil, err
	}

	// Read the member names from the channel.
	members := []*member{}
	for mountEntry := range memberChan {
		if mountEntry.Error != nil {
			return nil, fmt.Errorf("Error while getting member: %v\n", mountEntry.Error)
		} else {
			if member, err := cr.newMember(mountEntry.Name); err != nil {
				// Member has disconnected.
			} else {
				members = append(members, member)
			}
		}
	}
	sort.Sort(byName(members))
	return members, nil
}

// broadcastMessage sends a message to all members in the channel.
func (cr *channel) broadcastMessage(messageText string) error {
	members, err := cr.getMembers()
	if err != nil {
		return err
	}

	for _, member := range members {
		// TODO(nlacasse): Sending messages async means they might get sent out of
		// order. Consider either sending them sync or maintain a queue.
		// TODO(sadovsky): We should bind to (and call SendMessage against) the
		// mounttable-based name, not the actual member endpoint.
		go cr.sendMessageTo(member, messageText)
	}
	return nil
}

func (cr *channel) sendMessageTo(member *member, messageText string) {
	ctx, cancel := context.WithTimeout(cr.ctx, 5*time.Second)
	defer cancel()

	s := service.ChatClient(member.Path)

	if err := s.SendMessage(ctx, messageText); err != nil {
		return // member has disconnected.
	}
}
