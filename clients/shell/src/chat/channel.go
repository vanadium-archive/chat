package main

// channel holds logic for finding and communicating with members of a
// channel.
//
// Usage:
//  // Construct a new channel.
//  cr := newChannel(ctx, mounttable, proxy, "path/to/channel/name")
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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/ipc"
	"v.io/v23/naming"
	"v.io/v23/options"
	"v.io/v23/security"
	mt "v.io/v23/services/mounttable"
	"v.io/v23/services/security/access"

	_ "v.io/x/ref/profiles/roaming"

	"chat/vdl"
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

var _ vdl.ChatServerMethods = (*chatServerMethods)(nil)

func newChatServerMethods(messages chan<- message) *chatServerMethods {
	return &chatServerMethods{
		messages: messages,
	}
}

// SendMessage is called by clients to send a message to the server.
func (cs *chatServerMethods) SendMessage(call ipc.ServerCall, IncomingMessage string) error {
	var sender Sender
	remoteb, _ := security.BlessingNames(call.Context(), security.CallSideRemote)
	sender = Sender(remoteb)
	cs.messages <- message{
		Sender:    sender,
		Text:      IncomingMessage,
		Timestamp: time.Now(),
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
	// Channel that emits incoming messages.
	messages chan message
	// Cached list of channel members.
	members []*member
}

func newChannel(ctx *context.T, mounttable, proxy, path string) (*channel, error) {
	// Set the namespace root to the mounttable passed on the command line.
	newCtx, _, err := v23.SetNewNamespace(ctx, mounttable)
	if err != nil {
		return nil, err
	}

	// Set the proxy that will be used to listen.
	listenSpec := v23.GetListenSpec(ctx)
	listenSpec.Proxy = proxy

	s, err := v23.NewServer(newCtx)
	if err != nil {
		return nil, err
	}

	messages := make(chan message)

	return &channel{
		chatServerMethods: newChatServerMethods(messages),
		messages:          messages,
		path:              path,
		ctx:               newCtx,
		server:            s,
	}, nil
}

// openAuthorizer allows RPCs from all clients.
// TODO(nlacasse): Write a more strict authorizer once we have a better
// understanding of ACLs and identities, and once javascript supports similar
// functionality.
type openAuthorizer struct{}

func (o openAuthorizer) Authorize(*context.T) error {
	return nil
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

		// SetACL succeeded!  We now own the name.
		return name, nil
	}
	return "", fmt.Errorf("Error getting a locked name.  Tried %v times but did not succeed.", maxTries)
}

// join starts a chat server and mounts it in the channel path.
func (cr *channel) join() error {
	if _, err := cr.server.Listen(v23.GetListenSpec(cr.ctx)); err != nil {
		return err
	}
	serverChat := vdl.ChatServer(cr.chatServerMethods)

	name, err := cr.getLockedName()
	if err != nil {
		return err
	}

	if err := cr.server.Serve(name, serverChat, openAuthorizer{}); err != nil {
		return err
	}

	return nil
}

func (cr *channel) leave() error {
	// Stop serving.
	cr.server.Stop()

	// Get the names we are mounted at.  Should only be one.
	names := cr.server.Status().Mounts.Names()
	// Delete the ACL from the name and all sub-names in the hierarchy.
	ns := v23.GetNamespace(cr.ctx)
	for _, name := range names {
		if err := ns.Delete(cr.ctx, name, true); err != nil {
			return err
		}
	}

	return nil
}

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
		case *naming.MountEntry:
			if len(v.Servers) == 0 {
				// No servers mounted at that name, only a
				// lonely ACL.  Safe to ignore.
				// TODO(nlacasse): Should there be a time-limit
				// on ACLs in the namespace?  Seems like we'll
				// have an ACL graveyard before too long.
				continue
			}

			member := cr.newMember(v.Servers[0].BlessingPatterns, v.Name)
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

func (cr *channel) sendMessageTo(member *member, messageText string) {
	ctx, cancel := context.WithTimeout(cr.ctx, 5*time.Second)
	defer cancel()

	s := vdl.ChatClient(member.Path)

	// The AllowedServersPolicy options require that the server matches the
	// blessings we got when we globbed it.
	opts := make([]ipc.CallOpt, len(member.Blessings))
	for i, blessing := range member.Blessings {
		opts[i] = options.AllowedServersPolicy{security.BlessingPattern(blessing)}
	}

	if err := s.SendMessage(ctx, messageText, opts...); err != nil {
		return // member has disconnected.
	}
}
