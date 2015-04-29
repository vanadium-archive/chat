#!/bin/bash
# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

readonly DIR=$(dirname $0)
export V23_CREDENTIALS="${DIR}/credentials"
if [[ -d "${DIR}/credentials" ]]
then
	agentd "${DIR}/go/bin/chat"
else
	agentd bash -c "principal seekblessings && ${DIR}/go/bin/chat"
fi
