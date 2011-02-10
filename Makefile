# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
include $(GOROOT)/src/Make.inc

TARG=http/session
GOFILES=\
	session.go\

include $(GOROOT)/src/Make.pkg
