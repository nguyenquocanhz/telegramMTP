package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nguyenquocanhz/telegramMTP/secret"
)

// secretCLI manages the encrypted store. The passphrase comes from
// SECRET_MASTER_KEY, resolved like any other secret.
func secretCLI(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: natterbot secret set KEY | list | rm KEY")
	}

	// A wrong master key makes secret.New fail, which is the error the
	// user needs to see for every subcommand.
	mgr, err := secret.New(secret.Options{})
	if err != nil {
		return err
	}
	master, err := mgr.Require(secret.MasterKeyName)
	if err != nil {
		return fmt.Errorf("%w\nset it first, e.g. in PowerShell: $env:%s = '<passphrase>'", err, secret.MasterKeyName)
	}
	store, err := secret.OpenStore(mgr.StorePath(), master)
	if err != nil {
		return err
	}

	switch args[0] {
	case "set":
		if len(args) != 2 {
			return errors.New("usage: natterbot secret set KEY   (the value is read from stdin)")
		}
		key := args[1]
		// Reading from stdin keeps the value out of the shell history
		// and out of the process list.
		fmt.Fprintf(os.Stderr, "Value for %s: ", key)
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		value := strings.TrimSpace(line)
		if value == "" {
			if err != nil {
				return fmt.Errorf("no value read from stdin: %w", err)
			}
			return errors.New("empty value, nothing stored")
		}
		store.Set(key, value)
		if err := store.Save(); err != nil {
			return err
		}
		fmt.Printf("stored %s in %s\n", key, mgr.StorePath())

	case "list":
		keys := store.Keys()
		if len(keys) == 0 {
			fmt.Printf("%s is empty\n", mgr.StorePath())
			return nil
		}
		for _, key := range keys {
			value, _ := store.Get(key)
			fmt.Printf("%-20s %s\n", key, secret.Mask(value))
		}

	case "rm":
		if len(args) != 2 {
			return errors.New("usage: natterbot secret rm KEY")
		}
		if !store.Delete(args[1]) {
			return fmt.Errorf("%s is not in %s", args[1], mgr.StorePath())
		}
		if err := store.Save(); err != nil {
			return err
		}
		fmt.Printf("removed %s\n", args[1])

	default:
		return fmt.Errorf("unknown secret command %q (want set, list or rm)", args[0])
	}
	return nil
}
