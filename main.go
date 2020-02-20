package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/goccy/go-yaml"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
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
  modify-user-data    Modify user-data.
  make-iso            Make an ISO image

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
	case "modify-user-data":
		err = runModifyUserDataCmd(args[1:])
	case "make-iso":
		err = runMakeISOCmd(args[1:])
	default:
		flag.Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
		var roerr *requiredOptionError
		if errors.As(err, &roerr) {
			roerr.fs.Usage()
			return 2
		}
		return 1
	}
	return 0
}

const modifyUserDataCmdUsage = `Usage: %s modify-user-data [options]

options:
`

func runModifyUserDataCmd(args []string) error {
	fs := flag.NewFlagSet("modify-user-data", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), modifyUserDataCmdUsage, cmdName)
		fs.PrintDefaults()
	}
	in := fs.String("in", "", "input user-data yaml file. required.")
	out := fs.String("out", "", "output user-data yaml file. required.")
	inputPasswd := fs.Bool("passwd", false, "show prompt to input default user password. optional.")
	pubKeyFilename := fs.String("pub-key", "", "add ssh public key to ssh_authorized_keys. optional.")
	fs.Parse(args)

	if *in == "" {
		return newRequiredOptionError(fs, "in")
	}
	if *out == "" {
		return newRequiredOptionError(fs, "out")
	}

	var password []byte
	if *inputPasswd {
		var err error
		password, err = readPassword()
		if err != nil {
			return err
		}
	}

	inData, err := ioutil.ReadFile(*in)
	if err != nil {
		return err
	}
	var userData map[string]interface{}
	if err := yaml.Unmarshal(inData, &userData); err != nil {
		return err
	}

	if *inputPasswd {
		const cost = 11
		passHash, err := bcrypt.GenerateFromPassword(password, cost)
		if err != nil {
			return err
		}
		userData["password"] = string(passHash)
	}

	if *pubKeyFilename != "" {
		pubKey, err := ioutil.ReadFile(*pubKeyFilename)
		if err != nil {
			return err
		}

		var keys []interface{}
		sshAuthorizedKeys, ok := userData["ssh_authorized_keys"]
		if ok {
			keys, ok = sshAuthorizedKeys.([]interface{})
			if !ok {
				fmt.Printf("sshAuthorizedKeys type=%T\n", sshAuthorizedKeys)
				return errors.New("ssh_authorized_keys must have a string value")
			}
		}
		keys = append(keys, string(pubKey))
		userData["ssh_authorized_keys"] = keys
	}

	file, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "#cloud-config")

	b, err := yaml.Marshal(userData)
	if err != nil {
		return err
	}
	if _, err := file.Write(b); err != nil {
		return err
	}
	return nil
}

func readPassword() ([]byte, error) {
	for {
		if _, err := os.Stdout.Write([]byte("Password: ")); err != nil {
			return nil, err
		}
		passwd1, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}

		if _, err := os.Stdout.Write([]byte("\nConfirm password: ")); err != nil {
			return nil, err
		}
		passwd2, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}

		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			return nil, err
		}

		if bytes.Equal(passwd1, passwd2) {
			return passwd1, nil
		}
	}
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
	networkConfigFilename := fs.String("network-config", "", "input network-config yaml file. optional.")
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

	if *networkConfigFilename != "" {
		networkConfig, err := ioutil.ReadFile(*networkConfigFilename)
		if err != nil {
			return err
		}
		if err := addFileToISO(isoFS, "network-config", networkConfig); err != nil {
			return err
		}
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
