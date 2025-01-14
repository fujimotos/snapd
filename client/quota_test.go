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

package client_test

import (
	"encoding/json"
	"io/ioutil"

	"github.com/snapcore/snapd/client"
	"gopkg.in/check.v1"
)

func (cs *clientSuite) TestCreateQuotaGroupInvalidName(c *check.C) {
	err := cs.cli.EnsureQuota("", "", nil, 0)
	c.Check(err, check.ErrorMatches, `cannot create or update quota group without a name`)
}

func (cs *clientSuite) TestEnsureQuotaGroup(c *check.C) {
	cs.rsp = `{
		"type": "sync",
		"status-code": 200
	}`

	c.Assert(cs.cli.EnsureQuota("foo", "bar", []string{"snap-a", "snap-b"}, 1001), check.IsNil)
	c.Check(cs.req.Method, check.Equals, "POST")
	c.Check(cs.req.URL.Path, check.Equals, "/v2/quotas")
	body, err := ioutil.ReadAll(cs.req.Body)
	c.Assert(err, check.IsNil)
	var req map[string]interface{}
	err = json.Unmarshal(body, &req)
	c.Assert(err, check.IsNil)
	c.Assert(req, check.DeepEquals, map[string]interface{}{
		"action":     "ensure",
		"group-name": "foo",
		"parent":     "bar",
		"snaps":      []interface{}{"snap-a", "snap-b"},
		"max-memory": float64(1001),
	})
}

func (cs *clientSuite) TestEnsureQuotaGroupError(c *check.C) {
	cs.status = 500
	cs.rsp = `{"type": "error"}`
	err := cs.cli.EnsureQuota("foo", "bar", []string{"snap-a"}, 1)
	c.Check(err, check.ErrorMatches, `cannot create or update quota group: server error: "Internal Server Error"`)
}

func (cs *clientSuite) TestGetQuotaGroupInvalidName(c *check.C) {
	_, err := cs.cli.GetQuotaGroup("")
	c.Assert(err, check.ErrorMatches, `cannot get quota group without a name`)
}

func (cs *clientSuite) TestGetQuotaGroup(c *check.C) {
	cs.rsp = `{
		"type": "sync",
		"status-code": 200,
		"result": {"group-name":"foo", "parent":"bar", "subgroups":["foo-subgrp"], "snaps":["snap-a"], "max-memory":999}
	}`

	grp, err := cs.cli.GetQuotaGroup("foo")
	c.Assert(err, check.IsNil)
	c.Check(cs.req.Method, check.Equals, "GET")
	c.Check(cs.req.URL.Path, check.Equals, "/v2/quotas/foo")
	c.Check(grp, check.DeepEquals, &client.QuotaGroupResult{
		GroupName: "foo",
		Parent:    "bar",
		Subgroups: []string{"foo-subgrp"},
		MaxMemory: 999,
		Snaps:     []string{"snap-a"},
	})
}

func (cs *clientSuite) TestGetQuotaGroupError(c *check.C) {
	cs.status = 500
	cs.rsp = `{"type": "error"}`
	_, err := cs.cli.GetQuotaGroup("foo")
	c.Check(err, check.ErrorMatches, `cannot get quota group: server error: "Internal Server Error"`)
}

func (cs *clientSuite) TestRemoveQuotaGroup(c *check.C) {
	cs.rsp = `{
		"type": "sync",
		"status-code": 200
	}`

	err := cs.cli.RemoveQuotaGroup("foo")
	c.Assert(err, check.IsNil)
	c.Check(cs.req.Method, check.Equals, "POST")
	c.Check(cs.req.URL.Path, check.Equals, "/v2/quotas")
	body, err := ioutil.ReadAll(cs.req.Body)
	c.Assert(err, check.IsNil)
	var req map[string]interface{}
	err = json.Unmarshal(body, &req)
	c.Assert(err, check.IsNil)
	c.Assert(req, check.DeepEquals, map[string]interface{}{
		"action":     "remove",
		"group-name": "foo",
	})
}

func (cs *clientSuite) TestRemoveQuotaGroupError(c *check.C) {
	cs.status = 500
	cs.rsp = `{"type": "error"}`
	err := cs.cli.RemoveQuotaGroup("foo")
	c.Check(err, check.ErrorMatches, `cannot remove quota group: server error: "Internal Server Error"`)
}
