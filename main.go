package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/goccy/go-yaml"
)

type requiredOptionError struct {
	fs     *flag.FlagSet
	option string
}

func newRequiredOptionError(fs *flag.FlagSet, option string) *requiredOptionError {
	return &requiredOptionError{fs: fs, option: option}
}

func (e *requiredOptionError) Error() string {
	return fmt.Sprintf("option -%s is required.", e.option)
}

const globalUsage = `Usage: %s <subcommand> [options]

subcommands:
  add-ssh-key    Add ssh key to user-data yaml.
  make-iso       Make an ISO image from user-data yaml.

Run %s <subcommand> -h to show help for subcommand.
`

func main() {
	os.Exit(run())
}

var cmdName = os.Args[0]

func run() int {
	flag.Usage = func() {
		fmt.Printf(globalUsage, cmdName, cmdName)
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return 2
	}

	var err error
	switch args[0] {
	case "add-ssh-key":
		err = runAddSshKeyCmd(args[1:])
	case "make-iso":
		err = runMakeISOCmd(args[1:])
	default:
		flag.Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err.Error())
		var roerr *requiredOptionError
		if errors.As(err, &roerr) {
			roerr.fs.Usage()
			return 2
		}
		return 1
	}
	return 0
}

const addSshKeyCmdUsage = `Usage: %s add-ssh-key [options]

options:
`

func runAddSshKeyCmd(args []string) error {
	fs := flag.NewFlagSet("add-ssh-key", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), addSshKeyCmdUsage, cmdName)
		fs.PrintDefaults()
	}
	in := fs.String("in", "", "input user-data yaml file. required.")
	out := fs.String("out", "", "output user-data yaml file. required.")
	privKeyFilename := fs.String("priv", "", "user ssh private key. required.")
	pubKeyFilename := fs.String("pub", "", "user ssh public key. required.")
	fs.Parse(args)

	if *in == "" {
		return newRequiredOptionError(fs, "in")
	}
	if *out == "" {
		return newRequiredOptionError(fs, "out")
	}
	if *privKeyFilename == "" {
		return newRequiredOptionError(fs, "priv")
	}
	if *pubKeyFilename == "" {
		return newRequiredOptionError(fs, "pub")
	}

	type WriteFile struct {
		Path    string `yaml:"path"`
		Content string `yaml:"content"`

		// NOTE: We cannot have Owner here
		// See https://bugs.launchpad.net/cloud-init/+bug/1486113
		// Owner string `yaml:"owner"`

		Permissions string `yaml:"permissions"`
	}

	type Chpasswd struct {
		Expire bool `yaml:"expire"`
	}

	type AptRepo struct {
		Arches []string `yaml:"arches"`
		URI    string   `yaml:"uri"`
	}

	type UserData struct {
		Locale                  string             `yaml:"locale"`
		Timezone                string             `yaml:"timezone"`
		PackageUpgrade          bool               `yaml:"package_upgrade"`
		PackageRebootIfRequired bool               `yaml:"package_reboot_if_required"`
		Apt                     map[string]AptRepo `yaml:"apt"`
		Password                string             `yaml:"password"`
		Chpasswd                Chpasswd           `yaml:"chpasswd"`
		SSHAuthorizeKeys        []string           `yaml:"ssh_authorized_keys"`
		WriteFiles              []WriteFile        `yaml:"write_files"`
	}

	var data UserData
	inData, err := ioutil.ReadFile(*in)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(inData, &data); err != nil {
		return err
	}

	privKey, err := ioutil.ReadFile(*privKeyFilename)
	if err != nil {
		return err
	}
	pubKey, err := ioutil.ReadFile(*pubKeyFilename)
	if err != nil {
		return err
	}

	data.SSHAuthorizeKeys = append(data.SSHAuthorizeKeys, string(pubKey))
	data.WriteFiles = []WriteFile{
		{
			Path:        "/priv_key",
			Content:     string(privKey),
			Permissions: "0400",
		},
		{
			Path:        "/pub_key",
			Content:     string(pubKey),
			Permissions: "0600",
		},
	}

	file, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "#cloud-config")

	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := file.Write(b); err != nil {
		return err
	}
	return nil
}

const makeISOCmdUsage = `Usage: %s make-iso [options]

options:
`

func runMakeISOCmd(args []string) error {
	fs := flag.NewFlagSet("make-iso", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), makeISOCmdUsage, cmdName)
		fs.PrintDefaults()
	}
	userDataFilename := fs.String("user-data", "", "input user-data yaml file. required.")
	metaDataFilename := fs.String("meta-data", "", "input meta-data yaml file. optional.")
	out := fs.String("out", "", "output ISO image file. required.")
	fs.Parse(args)

	if *userDataFilename == "" {
		return newRequiredOptionError(fs, "user-data")
	}
	if *out == "" {
		return newRequiredOptionError(fs, "out")
	}

	file, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer file.Close()

	isoFS, err := iso9660.Create(file, 0, 0, 2048)
	if err != nil {
		return err
	}

	if err = isoFS.Mkdir("/"); err != nil {
		return err
	}

	metaData := []byte{}
	if *metaDataFilename != "" {
		metaData, err = ioutil.ReadFile(*metaDataFilename)
		if err != nil {
			return err
		}
	}
	if err := addFileToISO(isoFS, "meta-data", metaData); err != nil {
		return err
	}

	userData, err := ioutil.ReadFile(*userDataFilename)
	if err != nil {
		return err
	}
	if err := addFileToISO(isoFS, "user-data", userData); err != nil {
		return err
	}

	err = isoFS.Finalize(iso9660.FinalizeOptions{
		RockRidge:        true,
		VolumeIdentifier: "cidata",
	})
	if err != nil {
		return err
	}

	return nil
}

func addFileToISO(fs *iso9660.FileSystem, filename string, data []byte) error {
	dest, err := fs.OpenFile("/"+filename, os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return err
	}

	if _, err := dest.Write(data); err != nil {
		return err
	}

	return nil
}
