/*  Copyright (C) 2017 Ryan Clarke.
 *
 *  Copying and distribution of this file, with or without modification,
 *  are permitted in any medium without royalty provided the copyright
 *  notice and this notice are preserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var LocalDir = filepath.Join(os.Getenv("APPDATA"), "Guild Wars 2")
var LocalDat = filepath.Join(LocalDir, "Local.dat")

func main() {
	name := flag.String("name", "Local", "Profile name to load")
	n := flag.Int("n", 2, "Number of profile backups to keep")

	autologin := flag.Bool("autologin", true, "Log in automatically")
	loadinfo := flag.Bool("loadinfo", true, "Show map load diagnostics")
	image := flag.Bool("image", false, "Download updates and exit")
	email := flag.String("email", "", "Email address for login")
	password := flag.String("password", "", "Password for login")

	flag.Parse()
	options := flag.Args()
	if *autologin {
		options = append(options, "-autologin")
	}
	if *loadinfo {
		options = append(options, "-maploadinfo")
	}
	if *image {
		options = append(options, "-image")
	}
	if *email != "" && *password != "" {
		options = append([]string{
			"-email=" + *email,
			"-password=" + *password,
			"-nopatchui",
		}, options...)
	}
	profile := Profile{Name: *name, Preserve: *n, Options: options}

	if strings.ToLower(profile.Name) != "local" {
		// Load selected profile, and unload it when finished
		if err := profile.Load(); err != nil {
			exitPrompt(err)
		}
		defer time.Sleep(1 * time.Second)
		defer profile.Unload()
	}

	if err := profile.Run(); err != nil {
		exitPrompt(err)
	}
}

func startGw2(options []string) error {
	for _, gw2exe := range []string{
		filepath.Join(os.Getenv("PROGRAMFILES(x86)"), "Guild Wars 2", "Gw2-64.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(x86)"), "Guild Wars 2", "Gw2.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Guild Wars 2", "Gw2-64.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Guild Wars 2", "Gw2.exe"),
	} {
		if _, err := os.Stat(gw2exe); !os.IsNotExist(err) {
			log.Println("launching Guild Wars 2")
			defer log.Println("exiting Guild Wars 2")
			return exec.Command(gw2exe, options...).Run()
		}
	}
	return errors.New("launcher not found - is Guild Wars 2 installed?")
}

type Profile struct {
	Name     string
	Options  []string
	Preserve int
	path     string
}

// Find the path to the Guild Wars 2 launcher and run it with the given options
func (this *Profile) Run() error {
	for _, gw2exe := range []string{
		filepath.Join(os.Getenv("PROGRAMFILES(x86)"), "Guild Wars 2", "Gw2-64.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(x86)"), "Guild Wars 2", "Gw2.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Guild Wars 2", "Gw2-64.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Guild Wars 2", "Gw2.exe"),
	} {
		if _, err := os.Stat(gw2exe); !os.IsNotExist(err) {
			log.Println("launching Guild Wars 2")
			defer log.Println("exiting Guild Wars 2")
			return exec.Command(gw2exe, this.Options...).Run()
		}
	}
	return errors.New("launcher not found - is Guild Wars 2 installed?")
}

func (this *Profile) Path() string {
	if this.path != "" {
		return this.path
	}
	return filepath.Join(LocalDir, this.Name+".dat")
}

func (this *Profile) Load() error {
	if _, err := os.Stat(LocalDat); !os.IsNotExist(err) {
		log.Println("backing up original profile")
		if err := os.Rename(LocalDat, LocalDat+".bak"); err != nil {
			return err
		}
	}

	if _, err := os.Stat(this.Path()); !os.IsNotExist(err) {
		log.Printf("loading profile for %s\n", this.Name)
		return SimpleCopy(this.Path(), LocalDat)
	}
	log.Printf("creating profile for %s\n", this.Name)
	return nil
}

func (this *Profile) Unload() error {
	if err := this.RollBackups(); err != nil {
		return err
	}

	log.Printf("unloading profile for %s\n", this.Name)
	if err := SimpleCopy(LocalDat, this.Path()); err != nil {
		return err
	}

	if _, err := os.Stat(LocalDat + ".bak"); !os.IsNotExist(err) {
		log.Println("restoring original profile")
		return os.Rename(LocalDat+".bak", LocalDat)
	}
	return nil
}

func (this *Profile) RollBackups() error {
	if this.Preserve < 1 {
		return nil
	}

	log.Printf("managing profile backups (max %d)\n", this.Preserve)
	if err := os.RemoveAll(this.getBackupName(this.Preserve - 1)); err != nil {
		return err
	}
	for i := this.Preserve - 1; i > 0; i-- {
		backupOld := this.getBackupName(i - 1)
		backupNew := this.getBackupName(i)
		if _, err := os.Stat(backupOld); !os.IsNotExist(err) {
			if err := os.Rename(backupOld, backupNew); err != nil {
				return err
			}
		}
	}
	return os.Rename(this.Path(), this.getBackupName(0))
}

func (this *Profile) getBackupName(i int) string {
	return fmt.Sprintf("%s-backup.%d", this.Path(), i)
}

func SimpleCopy(src, dst string) error {
	srcf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcf.Close()

	dstf, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstf.Close()

	_, err = io.Copy(dstf, srcf)
	return err
}

func exitPrompt(err error) {
	if err != nil {
		log.Println(err)
	}
	fmt.Println("\npress the [ENTER] key to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
