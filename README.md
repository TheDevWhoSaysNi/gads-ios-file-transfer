# GADS File Transfer

Send a file from a computer to an iPhone. Photos and videos show up in **Photos** and in **Files**. Everything else shows up in **Files**, under **On My iPhone → GADS File Transfer**.

The phone needs the small app in [`ios/`](ios/) installed once. The first launch asks for permission to add to Photos. After that, a send does not ask again.

This does not depend on a particular web console, a Linux hub, or a Mac mini. The computer that has the USB cable runs the tool. That computer can be macOS, Linux, or Windows.

## What you install on the phone

On a Mac, from `ios/`:

```sh
./build-ipa.sh
```

That writes `ios/GADSFileTransfer-unsigned.ipa`. Sign it with the same certificate and provisioning profile you already use for WebDriverAgent ([zsign](https://github.com/zhkl0228/zsign) is the usual tool). The profile has to include the phone, and the bundle id is `com.gads.filetransfer`.

```sh
ios install --path=GADSFileTransfer-signed.ipa
ios launch com.gads.filetransfer
```

Tap **Allow** when the phone asks to add to Photos. `ios` is [go-ios](https://github.com/danielpaulus/go-ios). On iOS 17 and newer, start its tunnel first (`ios tunnel start`) if GADS is not already running on that computer.

## Send a file

Build the host tool, on the computer the phone is plugged into:

```sh
go build -o gads-file-transfer ./cmd/gads-file-transfer
```

List phones:

```sh
./gads-file-transfer devices
```

Copy one file:

```sh
./gads-file-transfer push --udid PHONE_UDID photo.jpg
```

Or open a page in a browser:

```sh
./gads-file-transfer serve
```

Then open `http://127.0.0.1:8787` on that computer. Pick the phone, pick a file, send.

### From another computer, or through a tunnel

The page listens on localhost unless you say otherwise. To open it from a phone or laptop on the same network:

```sh
./gads-file-transfer serve --listen 0.0.0.0:8787 --token 'a long random secret'
```

A public address without `--token` is refused. Put the token in the page when it asks.

A Cloudflare tunnel (or any other tunnel) to `http://127.0.0.1:8787` still needs a token. The tunnel connects from the same machine, so the program cannot tell a tunnel apart from you:

```sh
./gads-file-transfer serve --token 'a long random secret'
cloudflared tunnel --url http://127.0.0.1:8787
```

Linux needs `usbmuxd`. Windows needs Apple Mobile Device Support (iTunes or the Apple Devices app) and, for the go-ios tunnel, `wintun.dll`. Details are in the [go-ios readme](https://github.com/danielpaulus/go-ios).

## With a GADS provider

If [GADS](https://github.com/shamanec/GADS) is already running on the USB computer, let it launch the app. It already holds the iOS 17 tunnel and WebDriverAgent.

```sh
./gads-file-transfer serve --provider http://127.0.0.1:PROVIDER_PORT --token 'a long random secret'
```

`PROVIDER_PORT` is that provider's own port, not the hub. The same flag works on `push`.

For a hub and a web UI in front of many providers, teach the provider itself. The hub already forwards `POST /device/{udid}/files/push` to the provider that owns the phone. Android uses that route today. The iPhone branch is a short paste, written in [docs/gads-provider.md](docs/gads-provider.md). After that paste, the stock GADS upload and any custom console can send files to iPhones through the hub. No second web UI is required.

Do not point a second copy of this tool at a provider launch until the app is installed. The provider's normal app-launch helper restarts that phone's session when launch fails.

## What the person sees

The app comes to the front, copies new photos and videos into Photos, and leaves every file in its Files folder. With `--provider`, the phone then goes back to the home screen. Without a provider, switch back yourself. The page tells you where the file landed.

Sending the same bytes again does not create a second Photos item. A changed file does.
