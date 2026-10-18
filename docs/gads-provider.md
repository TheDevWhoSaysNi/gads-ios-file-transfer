# GADS provider

The stock provider accepts `POST /device/{udid}/files/push` for Android and returns an error for iPhone. Paste the function below into the provider, then call it from `PushFileToSharedStorage` in `provider/router/routes.go` when `platDev.GetOS() == "ios"`.

Use the device's existing go-ios entry. Do not call `IOSDevice.LaunchApp`. That method restarts the whole phone session when launch fails, which would drop WebDriverAgent.

```go
package devices

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielpaulus/go-ios/ios/afc"
	"github.com/danielpaulus/go-ios/ios/house_arrest"
	"github.com/danielpaulus/go-ios/ios/instruments"
)

const fileTransferBundleID = "com.gads.filetransfer"

// PushToFileTransfer copies a file into GADS File Transfer, opens the app so
// photos and videos are added to Photos, then returns to the home screen.
func (d *IOSDevice) PushToFileTransfer(localPath, fileName string) error {
	client, err := house_arrest.New(d.GoIOSDeviceEntry, fileTransferBundleID)
	if err != nil {
		return fmt.Errorf("GADS File Transfer is not installed: %w", err)
	}
	dest := fileTransferDest(client, fileName)
	status := fileTransferDest(client, ".gads-file-transfer.json")
	if err := client.Push(localPath, dest); err != nil {
		client.Close()
		return err
	}
	_ = client.Remove(status)
	client.Close()

	control, err := instruments.NewProcessControl(d.GoIOSDeviceEntry)
	if err != nil {
		return fmt.Errorf("file is in Files, but the app did not open: %w", err)
	}
	defer control.Close()
	if _, err := control.LaunchAppWithArgs(fileTransferBundleID, nil, nil, map[string]any{"KillExisting": 1}); err != nil {
		return fmt.Errorf("file is in Files, but the app did not open: %w", err)
	}

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		reader, err := house_arrest.New(d.GoIOSDeviceEntry, fileTransferBundleID)
		if err != nil {
			continue
		}
		tmp := localPath + ".status.json"
		err = reader.Pull(fileTransferDest(reader, ".gads-file-transfer.json"), tmp)
		reader.Close()
		if err != nil {
			continue
		}
		raw, readErr := os.ReadFile(tmp)
		os.Remove(tmp)
		if readErr == nil && strings.Contains(string(raw), `"done"`) && strings.Contains(string(raw), fileName) {
			break
		}
	}

	url := fmt.Sprintf("http://localhost:%s/wda/homescreen", d.GetWDAPort())
	res, err := http.Post(url, "application/json", nil)
	if err == nil {
		res.Body.Close()
	}
	return nil
}

func fileTransferDest(client *afc.Client, name string) string {
	if _, err := client.Stat("/Documents"); err == nil {
		return "/Documents/" + name
	}
	return "/" + name
}
```

In `PushFileToSharedStorage`, replace the iOS rejection with:

```go
if platDev.GetOS() == "ios" {
	iosDev, ok := platDev.(*devices.IOSDevice)
	if !ok {
		api.BadRequest(c, "Device is not an iPhone")
		return
	}
	// save the upload to tempPath the same way Android does, then:
	if err := iosDev.PushToFileTransfer(tempPath, file.Filename); err != nil {
		api.InternalError(c, err.Error())
		return
	}
	api.OKMessage(c, "File sent")
	return
}
```

After that, any client of the hub can use the existing files push route. A custom web UI is optional. The GADS hub UI's Android upload hits the same route.

This is the change to contribute back to [shamanec/GADS](https://github.com/shamanec/GADS). This repo and that patch are both AGPL-3.0, the same license as GADS.
