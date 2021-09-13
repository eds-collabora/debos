/*
Pacstrap Action

Construct the target rootfs with pacstrap tool.

Yaml syntax:
 - action: pacstrap
   mirror: <url with placeholders>
   repositories: <list of repositories>

Mandatory properties:

 - mirror -- the full url for the repository, with placeholders for
   $arch and $repo as needed, as would be found in mirrorlist

Optional properties:
 - repositories -- list of repositories to use for packages selection.
   Properties for repositories are described below.

Yaml syntax for repositories:

 repositories:
   - name: repository name
     siglevel: signature checking settings (optional)
*/
package actions

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/go-debos/debos"
)

type PacstrapAction struct {
	debos.BaseAction `yaml:",inline"`
	ConfigFile       string
	MirrorFile       string
}

func (d *PacstrapAction) listConfigFiles(context *debos.DebosContext) []string {
	files := []string{}
	files = append(files, debos.CleanPathAt(d.ConfigFile, context.RecipeDir))
	if d.MirrorFile != "" {
		files = append(files, debos.CleanPathAt(d.MirrorFile, context.RecipeDir))
	}
	return files
}

func (d *PacstrapAction) Verify(context *debos.DebosContext) error {
	if d.ConfigFile == "" {
		return fmt.Errorf("No config file provided.")
	}
	files := d.listConfigFiles(context)
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (d *PacstrapAction) Run(context *debos.DebosContext) error {
	d.LogStart()

	// Mirror list for pacstrap. Note that this is copied into the
	// chroot, so will persist after pacstrap, whereas the
	// pacman.conf file will not (it will be replaced with the
	// stock pacman.conf).
	if d.MirrorFile != "" {
		mirrorListPath := filepath.Join(context.Rootdir, "/etc/pacman.d/mirrorlist")
		if err := debos.CopyFile(d.ConfigFile, mirrorListPath, 0644); err != nil {
			return err
		}
	}

	// Run pacman-key Note that the host's pacman/gnupg secrets
	// are root-only, and we want to avoid running
	// fakemachine/debos as root. As such, explicitly run
	// pacman-key --init so that new set is generated.
	cmdline := []string{"pacman-key", "--config", d.ConfigFile, "--init", "--populate"}
	if err := (debos.Command{}.Run("Pacman-key", cmdline...)); err != nil {
		return fmt.Errorf("Couldn't init pacman keyring: %v", err)
	}

	// Run pacstrap
	cmdline = []string{"pacstrap", "-M", "-C", d.ConfigFile, context.Rootdir}
	if err := (debos.Command{}.Run("Pacstrap", cmdline...)); err != nil {
		log := path.Join(context.Rootdir, "var/log/pacman.log")
		_ = debos.Command{}.Run("pacstrap.log", "cat", log)
		return err
	}

	return nil
}
