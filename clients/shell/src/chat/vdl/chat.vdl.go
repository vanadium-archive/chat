// This file was auto-generated by the veyron vdl tool.
// Source: chat.vdl

package vdl

import (
	// VDL system imports
	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/ipc"
)

// ChatClientMethods is the client interface
// containing Chat methods.
type ChatClientMethods interface {
	// SendMessage sends a message to a user.
	SendMessage(ctx *context.T, text string, opts ...ipc.CallOpt) error
}

// ChatClientStub adds universal methods to ChatClientMethods.
type ChatClientStub interface {
	ChatClientMethods
	ipc.UniversalServiceMethods
}

// ChatClient returns a client stub for Chat.
func ChatClient(name string, opts ...ipc.BindOpt) ChatClientStub {
	var client ipc.Client
	for _, opt := range opts {
		if clientOpt, ok := opt.(ipc.Client); ok {
			client = clientOpt
		}
	}
	return implChatClientStub{name, client}
}

type implChatClientStub struct {
	name   string
	client ipc.Client
}

func (c implChatClientStub) c(ctx *context.T) ipc.Client {
	if c.client != nil {
		return c.client
	}
	return v23.GetClient(ctx)
}

func (c implChatClientStub) SendMessage(ctx *context.T, i0 string, opts ...ipc.CallOpt) (err error) {
	var call ipc.Call
	if call, err = c.c(ctx).StartCall(ctx, c.name, "SendMessage", []interface{}{i0}, opts...); err != nil {
		return
	}
	err = call.Finish()
	return
}

// ChatServerMethods is the interface a server writer
// implements for Chat.
type ChatServerMethods interface {
	// SendMessage sends a message to a user.
	SendMessage(ctx ipc.ServerContext, text string) error
}

// ChatServerStubMethods is the server interface containing
// Chat methods, as expected by ipc.Server.
// There is no difference between this interface and ChatServerMethods
// since there are no streaming methods.
type ChatServerStubMethods ChatServerMethods

// ChatServerStub adds universal methods to ChatServerStubMethods.
type ChatServerStub interface {
	ChatServerStubMethods
	// Describe the Chat interfaces.
	Describe__() []ipc.InterfaceDesc
}

// ChatServer returns a server stub for Chat.
// It converts an implementation of ChatServerMethods into
// an object that may be used by ipc.Server.
func ChatServer(impl ChatServerMethods) ChatServerStub {
	stub := implChatServerStub{
		impl: impl,
	}
	// Initialize GlobState; always check the stub itself first, to handle the
	// case where the user has the Glob method defined in their VDL source.
	if gs := ipc.NewGlobState(stub); gs != nil {
		stub.gs = gs
	} else if gs := ipc.NewGlobState(impl); gs != nil {
		stub.gs = gs
	}
	return stub
}

type implChatServerStub struct {
	impl ChatServerMethods
	gs   *ipc.GlobState
}

func (s implChatServerStub) SendMessage(ctx ipc.ServerContext, i0 string) error {
	return s.impl.SendMessage(ctx, i0)
}

func (s implChatServerStub) Globber() *ipc.GlobState {
	return s.gs
}

func (s implChatServerStub) Describe__() []ipc.InterfaceDesc {
	return []ipc.InterfaceDesc{ChatDesc}
}

// ChatDesc describes the Chat interface.
var ChatDesc ipc.InterfaceDesc = descChat

// descChat hides the desc to keep godoc clean.
var descChat = ipc.InterfaceDesc{
	Name:    "Chat",
	PkgPath: "chat/vdl",
	Methods: []ipc.MethodDesc{
		{
			Name: "SendMessage",
			Doc:  "// SendMessage sends a message to a user.",
			InArgs: []ipc.ArgDesc{
				{"text", ``}, // string
			},
		},
	},
}
