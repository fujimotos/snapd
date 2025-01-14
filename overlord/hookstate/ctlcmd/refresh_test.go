// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2021 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package ctlcmd_test

import (
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/overlord/hookstate"
	"github.com/snapcore/snapd/overlord/hookstate/ctlcmd"
	"github.com/snapcore/snapd/overlord/hookstate/hooktest"
	"github.com/snapcore/snapd/overlord/state"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/testutil"

	. "gopkg.in/check.v1"
)

type refreshSuite struct {
	testutil.BaseTest
	st          *state.State
	mockHandler *hooktest.MockHandler
}

var _ = Suite(&refreshSuite{})

func (s *refreshSuite) SetUpTest(c *C) {
	s.BaseTest.SetUpTest(c)
	dirs.SetRootDir(c.MkDir())
	s.AddCleanup(func() { dirs.SetRootDir("/") })
	s.st = state.New(nil)
	s.mockHandler = hooktest.NewMockHandler()
}

var refreshFromHookTests = []struct {
	args                []string
	base, restart       bool
	stdout, stderr, err string
	exitCode            int
}{{
	args: []string{"refresh", "--proceed", "--hold"},
	err:  "cannot use --proceed and --hold together",
}, {
	args: []string{"refresh", "--proceed"},
	err:  "not implemented yet",
}, {
	args: []string{"refresh", "--hold"},
	err:  "not implemented yet",
}, {
	args:   []string{"refresh", "--pending"},
	stdout: "pending: \nchannel: \nbase: false\nrestart: false\n",
}, {
	args:    []string{"refresh", "--pending"},
	base:    true,
	restart: true,
	stdout:  "pending: \nchannel: \nbase: true\nrestart: true\n",
}}

func (s *refreshSuite) TestRefreshFromHook(c *C) {
	s.st.Lock()
	task := s.st.NewTask("test-task", "my test task")
	setup := &hookstate.HookSetup{Snap: "snap1", Revision: snap.R(1), Hook: "gate-auto-refresh"}
	mockContext, err := hookstate.NewContext(task, s.st, setup, s.mockHandler, "")
	c.Check(err, IsNil)
	s.st.Unlock()

	for _, test := range refreshFromHookTests {
		mockContext.Lock()
		mockContext.Set("base", test.base)
		mockContext.Set("restart", test.restart)
		mockContext.Unlock()

		stdout, stderr, err := ctlcmd.Run(mockContext, test.args, 0)
		comment := Commentf("%s", test.args)
		if test.exitCode > 0 {
			c.Check(err, DeepEquals, &ctlcmd.UnsuccessfulError{ExitCode: test.exitCode}, comment)
		} else {
			if test.err == "" {
				c.Check(err, IsNil, comment)
			} else {
				c.Check(err, ErrorMatches, test.err, comment)
			}
		}

		c.Check(string(stdout), Equals, test.stdout, comment)
		c.Check(string(stderr), Equals, "", comment)
	}
}

func (s *refreshSuite) TestRefreshFromUnsupportedHook(c *C) {
	s.st.Lock()

	task := s.st.NewTask("test-task", "my test task")
	setup := &hookstate.HookSetup{Snap: "snap", Revision: snap.R(1), Hook: "install"}
	mockContext, err := hookstate.NewContext(task, s.st, setup, s.mockHandler, "")
	c.Check(err, IsNil)
	s.st.Unlock()

	_, _, err = ctlcmd.Run(mockContext, []string{"refresh"}, 0)
	c.Check(err, ErrorMatches, `can only be used from gate-auto-refresh hook`)
}

// TODO: support this case
func (s *refreshSuite) TestRefreshFromApp(c *C) {
	s.st.Lock()

	setup := &hookstate.HookSetup{Snap: "snap", Revision: snap.R(1)}
	mockContext, err := hookstate.NewContext(nil, s.st, setup, s.mockHandler, "")
	c.Check(err, IsNil)
	s.st.Unlock()

	_, _, err = ctlcmd.Run(mockContext, []string{"refresh"}, 0)
	c.Check(err, ErrorMatches, `cannot run outside of gate-auto-refresh hook`)
}

func (s *refreshSuite) TestRefreshRegularUserForbidden(c *C) {
	s.st.Lock()
	setup := &hookstate.HookSetup{Snap: "snap", Revision: snap.R(1)}
	s.st.Unlock()

	mockContext, err := hookstate.NewContext(nil, s.st, setup, s.mockHandler, "")
	c.Assert(err, IsNil)
	_, _, err = ctlcmd.Run(mockContext, []string{"refresh"}, 1000)
	c.Assert(err, ErrorMatches, `cannot use "refresh" with uid 1000, try with sudo`)
	forbidden, _ := err.(*ctlcmd.ForbiddenCommandError)
	c.Assert(forbidden, NotNil)
}
