// -*- Mode: Go; indent-tabs-mode: t -*-
// +build !excludeintegration

/*
 * Copyright (C) 2015, 2016 Canonical Ltd
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

package tests

import (
	"fmt"
	"path/filepath"
	"regexp"

	"gopkg.in/check.v1"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/integration-tests/testutils/cli"
	"github.com/snapcore/snapd/integration-tests/testutils/common"
	"github.com/snapcore/snapd/interfaces"
)

var _ = check.Suite(&snapOpSuite{})

type snapOpSuite struct {
	common.SnappySuite
}

func (s *snapOpSuite) TestInstallFailedIsUndone(c *check.C) {
	// make snap uninstallable
	snapName := "hello-world"
	subdirPath := filepath.Join("/snap", snapName, "current", "foo")
	_, err := cli.ExecCommandErr("sudo", "mkdir", "-p", subdirPath)
	c.Assert(err, check.IsNil)
	defer cli.ExecCommand(c, "sudo", "rm", "-rf", filepath.Dir(subdirPath))

	// try to install snap and see it fail
	_, err = cli.ExecCommandErr("sudo", "snap", "install", snapName)
	c.Assert(err, check.NotNil)

	// check undone and error in tasks
	output := cli.ExecCommand(c, "snap", "changes", snapName)
	expected := fmt.Sprintf(`(?ms).*\n(\d+) +Error.*Install "%s" snap\n$`, snapName)
	id := regexp.MustCompile(expected).FindStringSubmatch(output)[1]

	output = cli.ExecCommand(c, "snap", "change", id)

	type undoneCheckerFunc func(*check.C, string, string)
	for _, fn := range []undoneCheckerFunc{
		checkDownloadUndone,
		checkMountUndone,
		checkDataCopyUndone,
		checkSecProfilesSetupUndone} {
		fn(c, snapName, output)
	}
}

func checkDownloadUndone(c *check.C, snapName, output string) {
	expected := fmt.Sprintf(`(?ms).*Undone +.*Download snap %q from channel.*`, snapName)
	c.Assert(output, check.Matches, expected)
}

func checkMountUndone(c *check.C, snapName, output string) {
	expected := fmt.Sprintf(`(?ms).*Undone +.*Mount snap %q.*`, snapName)
	c.Assert(output, check.Matches, expected)

	// MountDir is removed /snap/<name>/<revision>
	checkEmptyGlob(c, filepath.Join(dirs.SnapSnapsDir, snapName, "[0-9]+"))

	// MountFile is removed /var/lib/snapd/snaps/<name>_<revision>.snap
	checkEmptyGlob(c, filepath.Join(dirs.SnapBlobDir,
		fmt.Sprintf("%s_%s.snap", snapName, "[0-9]+")))
}

func checkDataCopyUndone(c *check.C, snapName, output string) {
	expected := fmt.Sprintf(`(?ms).*Undone +.*Copy snap %q data\n.*`, snapName)
	c.Assert(output, check.Matches, expected)

	// DataHomeDir is removed /home/*/snap/<name>/<revision>
	checkEmptyGlob(c, filepath.Join(dirs.SnapDataHomeGlob, snapName, "[0-9]+"))

	// DataDir is removed /var/snap/<name>/<revision>
	checkEmptyGlob(c, filepath.Join(dirs.SnapDataDir, snapName, "[0-9]+"))
}

func checkSecProfilesSetupUndone(c *check.C, snapName, output string) {
	expected := fmt.Sprintf(`(?ms).*Undone +.*Setup snap %q security profiles.*`, snapName)
	c.Assert(output, check.Matches, expected)

	// security artifacts are removed for each backend: apparmor, seccomp, dbus, udev
	backends := map[string]string{
		dirs.SnapAppArmorDir:  interfaces.SecurityTagGlob(snapName),
		dirs.SnapSeccompDir:   interfaces.SecurityTagGlob(snapName),
		dirs.SnapBusPolicyDir: fmt.Sprintf("%s.conf", interfaces.SecurityTagGlob(snapName)),
		dirs.SnapUdevRulesDir: fmt.Sprintf("70-%s.rules", interfaces.SecurityTagGlob(snapName)),
	}
	for dir, glob := range backends {
		checkEmptyGlob(c, filepath.Join(dir, glob))
	}
}

func checkEmptyGlob(c *check.C, pattern string) {
	items, err := filepath.Glob(pattern)
	c.Assert(err, check.IsNil)
	c.Assert(items, check.IsNil)
}
