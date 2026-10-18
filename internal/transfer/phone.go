package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/afc"
	"github.com/danielpaulus/go-ios/ios/house_arrest"
	"github.com/danielpaulus/go-ios/ios/instruments"
)

// Device is one iPhone the host can see over USB.
type Device struct {
	UDID    string `json:"udid"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ListDevices returns iPhones currently connected to this computer.
func ListDevices() ([]Device, error) {
	list, err := ios.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	out := make([]Device, 0, len(list.DeviceList))
	for _, conn := range list.DeviceList {
		udid := conn.Properties.SerialNumber
		if udid == "" {
			continue
		}
		item := Device{UDID: udid, Name: udid}
		entry, err := ios.GetDevice(udid)
		if err == nil {
			if values, err := ios.GetValues(entry); err == nil {
				if values.Value.DeviceName != "" {
					item.Name = values.Value.DeviceName
				}
				item.Version = values.Value.ProductVersion
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// Send copies localPath onto the phone as remoteName, launches GADS File
// Transfer, and waits until the app reports that file.
func Send(ctx context.Context, udid, localPath, remoteName string, launch LaunchFunc) (Outcome, error) {
	name, err := SafeName(remoteName)
	if err != nil {
		return Outcome{}, err
	}
	entry, err := ios.GetDevice(udid)
	if err != nil {
		return Outcome{}, fmt.Errorf("find %s: %w", udid, err)
	}

	if err := push(entry, localPath, name); err != nil {
		return Outcome{}, err
	}
	if err := removeStatus(entry); err != nil {
		return Outcome{}, err
	}
	if err := launch(ctx, udid); err != nil {
		return Outcome{}, fmt.Errorf("file is in the app folder, but the app did not open: %w", err)
	}

	status, err := waitStatus(ctx, entry, name)
	if err != nil {
		return Outcome{}, err
	}
	return Describe(status, name), nil
}

// LaunchFunc opens the app. GADS should use the provider, which already holds
// the iOS 17 tunnel. A machine without GADS can use LaunchWithInstruments.
type LaunchFunc func(ctx context.Context, udid string) error

// LaunchWithInstruments opens the app through go-ios. On iOS 17+ this needs
// the tunnel (`ios tunnel start`) when GADS is not already providing one.
func LaunchWithInstruments(ctx context.Context, udid string) error {
	entry, err := ios.GetDevice(udid)
	if err != nil {
		return err
	}
	control, err := instruments.NewProcessControl(entry)
	if err != nil {
		return fmt.Errorf("%w (on iOS 17+ start the tunnel with `ios tunnel start`, or pass --provider)", err)
	}
	defer control.Close()
	_, err = control.LaunchAppWithArgs(BundleID, nil, nil, map[string]any{"KillExisting": 1})
	if err != nil {
		return fmt.Errorf("%w (on iOS 17+ start the tunnel with `ios tunnel start`, or pass --provider)", err)
	}
	return nil
}

func push(entry ios.DeviceEntry, localPath, name string) error {
	client, err := house_arrest.New(entry, BundleID)
	if err != nil {
		return fmt.Errorf("open %s (install and sign the app first): %w", BundleID, err)
	}
	defer client.Close()
	dest, err := documentFile(client, name)
	if err != nil {
		return err
	}
	if err := client.Push(localPath, dest); err != nil {
		return fmt.Errorf("copy %s onto the phone: %w", name, err)
	}
	return nil
}

func removeStatus(entry ios.DeviceEntry) error {
	client, err := house_arrest.New(entry, BundleID)
	if err != nil {
		return err
	}
	defer client.Close()
	dest, err := documentFile(client, StatusName)
	if err != nil {
		return err
	}
	_ = client.Remove(dest)
	return nil
}

func waitStatus(ctx context.Context, entry ios.DeviceEntry, name string) (Status, error) {
	deadline := time.Now().Add(90 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	var last error
	for {
		if err := ctx.Err(); err != nil {
			return Status{}, err
		}
		status, err := readStatus(entry)
		if err == nil && status.State == "done" && mentions(status, name) {
			return status, nil
		}
		last = err
		if time.Now().After(deadline) {
			if last != nil {
				return Status{}, fmt.Errorf("timed out waiting for the phone to finish: %w", last)
			}
			return Status{}, fmt.Errorf("timed out waiting for the phone to finish")
		}
		select {
		case <-ctx.Done():
			return Status{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func readStatus(entry ios.DeviceEntry) (Status, error) {
	client, err := house_arrest.New(entry, BundleID)
	if err != nil {
		return Status{}, err
	}
	defer client.Close()
	remote, err := documentFile(client, StatusName)
	if err != nil {
		return Status{}, err
	}
	tmp, err := os.CreateTemp("", "gads-file-transfer-*.json")
	if err != nil {
		return Status{}, err
	}
	tmpName := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpName)
	if err := client.Pull(remote, tmpName); err != nil {
		return Status{}, err
	}
	raw, err := os.ReadFile(tmpName)
	if err != nil {
		return Status{}, err
	}
	var status Status
	if err := json.Unmarshal(raw, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func mentions(status Status, name string) bool {
	for _, item := range status.Processed {
		if item.Name == name {
			return true
		}
	}
	for _, item := range status.Errors {
		if item.Name == name {
			return true
		}
	}
	return false
}

// documentFile returns the AFC path for a file in the app's Documents folder.
// A developer-signed app vends the whole container, so Documents is a subdirectory.
// VendDocuments (file sharing only) makes that folder the root.
func documentFile(client *afc.Client, name string) (string, error) {
	if _, err := client.Stat("/Documents"); err == nil {
		return "/Documents/" + name, nil
	}
	if _, err := client.Stat("Documents"); err == nil {
		return "Documents/" + name, nil
	}
	return "/" + name, nil
}
