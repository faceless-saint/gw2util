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

var ExecPath = "C:\\Program Files (x86)\\Guild Wars 2\\Gw2-64.exe"
var ProfileDir = filepath.Join(os.Getenv("APPDATA"), "Guild Wars 2")
var LoadedProfile = filepath.Join(ProfileDir, "Local.dat")

func main() {
	// Profile options
	name := flag.String("name", "Local", "Profile name to load")
	n := flag.Int("n", 2, "Number of profile backups to keep")

	// Guild Wars 2 launch options
	autologin := flag.Bool("autologin", true, "Log in automatically")
	loadinfo := flag.Bool("loadinfo", true, "Show map load diagnostics")
	image := flag.Bool("image", false, "Download updates and exit")
	email := flag.String("email", "", "Email address for login")
	password := flag.String("password", "", "Password for login")

	// Parse flags and initialize launch options
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

	// Profile name "local" skips loading/unloading
	//   (default behavior acts as a simple link to the GW2 executable)
    if strings.ToLower(profile.Name) != "local" {
		// Load selected profile, and unload it when finished
		if err := profile.LoadFile(); err != nil {
			Exit(err)
		}
		defer time.Sleep(1 * time.Second)
		defer profile.UnloadFile()
	}

	// Launch Guild Wars 2
	if err := LaunchGW2(profile.Options); err != nil {
		Exit(err)
	}
}

// LaunchGW2 starts the Guild Wars 2 main process
func LaunchGW2(options []string) error {
	log.Println("launching Guild Wars 2")
	return exec.Command(ExecPath, options...).Run()
}

type Profile struct {
	Name     string
	Options  []string
	Preserve int
	path     string
}

func (this *Profile) Path() string {
	if this.path != "" {
		return this.path
	}
	return filepath.Join(ProfileDir, this.Name+".dat")
}

// LoadFile backs up original state and loads the saved profile
func (this *Profile) LoadFile() error {
	// If a profile is already loaded, rename it as a backup
	if err := SimpleBackup(LoadedProfile); err != nil {
		return err
	}

	// Copy profile data file to loaded position
	log.Printf("loading profile for %s\n", this.Name)
	return SimpleCopy(this.Path(), LoadedProfile)
}

// UnloadFile saves changes to the profile and restores original state
func (this *Profile) UnloadFile() error {
	// Rotate backups of profile data
	if err := this.RollBackups(); err != nil {
		return err
	}

	// Copy profile data file from loaded position
	log.Printf("unloading profile for %s\n", this.Name)
	if err := SimpleCopy(LoadedProfile, this.Path()); err != nil {
		return err
	}

	// Restore the originally loaded profile
	return SimpleRestore(LoadedProfile)
}

// RollBackups manages rolling backups for the profile data
func (this *Profile) RollBackups() error {
	if this.Preserve < 1 {
		return nil
	}
	log.Printf("managing profile backups (max %d)\n", this.Preserve)
	if err := os.RemoveAll(this.getBackup(this.Preserve - 1)); err != nil {
		return err
	}
	for i := this.Preserve - 1; i > 0; i-- {
		if FileExists(this.getBackup(i - 1)) {
			if err := os.Rename(this.getBackup(i-1), this.getBackup(i)); err != nil {
				return err
			}
		}
	}
	return os.Rename(this.Path(), this.getBackup(0))
}

func (this *Profile) getBackup(i int) string {
	return fmt.Sprintf("%s.%d", this.Path(), i)
}

// SimpleBackup renames the given file as a backup
func SimpleBackup(file string) error {
	// If the file exists, rename it as a backup
	if FileExists(file) {
		log.Println("backing up original profile")
		return os.Rename(file, file+".bak")
	}
	return nil
}

// SimpleRestore reverses SimpleBackup, returning the original filename
func SimpleRestore(file string) error {
	// If the file backup exists, restore the original name
	if FileExists(file + ".bak") {
		log.Println("restoring original profile")
		return os.Rename(file+".bak", file)
	}
	return nil
}

// SimpleCopy copies the content of src into dst
func SimpleCopy(src, dst string) error {
	// Open source file for reading
	srcf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcf.Close()

	// Open destination file for writing
	dstf, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstf.Close()

	// Copy source file to destination, with mirrored writes to the hash
	_, err = io.Copy(dstf, srcf)
	return err
}

// FileExists returns true IFF the given file exists
func FileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

// Exit waits for user input and then returns with the given error
func Exit(err error) {
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