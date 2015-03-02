// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file was auto-generated via go generate.
// DO NOT UPDATE MANUALLY
package main

import "fmt"
import "testing"
import "os"

import "v.io/x/ref/lib/modules"
import "v.io/x/ref/lib/testutil"

func init() {
	modules.RegisterChild("FakeModulesMain", `FakeModulesMain is used to trick v23 test generate into generating
a modules TestMain.
TODO(mattr): This should be removed once v23 test generate is fixed.`, FakeModulesMain)
}

func TestMain(m *testing.M) {
	testutil.Init()
	if modules.IsModulesProcess() {
		if err := modules.Dispatch(); err != nil {
			fmt.Fprintf(os.Stderr, "modules.Dispatch failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	os.Exit(m.Run())
}
