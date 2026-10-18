# GADS File Transfer

Send a file from a computer to an iPhone. Photos and videos show up in **Photos** and in **Files**. Everything else shows up in **Files**, under **On My iPhone → GADS File Transfer**.

The phone needs the small app in [`ios/`](ios/) installed once. The first launch asks for permission to add to Photos. After that, a send does not ask again.

This does not depend on a particular web console, a Linux hub, or a Mac mini. The computer that has the USB cable runs the tool. That computer can be macOS, Linux, or Windows.

## Build, sign, and install

You build the unsigned app on a Mac, sign it once, then install that same signed IPA on every phone. Signing does not put the app on the phones by itself.

### 1. Build the unsigned IPA

Use a Mac with full Xcode, not the Command Line Tools alone. `xcode-select -p` should point at `Xcode.app`.

```sh
git clone https://github.com/TheDevWhoSaysNi/gads-ios-file-transfer.git
cd gads-ios-file-transfer/ios
chmod +x build-ipa.sh
./build-ipa.sh
```

That writes `ios/GADSFileTransfer-unsigned.ipa`.

### 2. Sign it once

Use [zsign](https://github.com/zhlynn/zsign) **1.1.2 or newer**. Check with `zsign -v`. Older builds, including 0.7, print `Signed OK!` and still produce a signature iOS rejects. `Can't find CodeSignature segment` followed by `Realloc CodeSignature space` and `Success!` is normal.

The password is the **.p12 export password**, not your Apple ID password.

The certificate inside the `.p12` has to be one of the certificates inside the `.mobileprovision`. A development account often has several certificates that all display as `Apple Development: Your Name`. A profile made only for the WebDriverAgent bundle id cannot sign `com.gads.filetransfer`. A wildcard profile (`TEAMID.*`) can, if it contains the same certificate as the `.p12`.

```sh
zsign \
  -k cert.p12 \
  -p 'P12_PASSWORD' \
  -m profile.mobileprovision \
  -o GADSFileTransfer.ipa \
  -b com.gads.filetransfer \
  GADSFileTransfer-unsigned.ipa
```

Check the result before installing a fleet:

```sh
mkdir -p /tmp/gads-check
unzip -q GADSFileTransfer.ipa -d /tmp/gads-check
codesign --verify --deep --strict /tmp/gads-check/Payload/GADSFileTransfer.app
codesign -d --verbose=2 /tmp/gads-check/Payload/GADSFileTransfer.app
```

`codesign --verify` should say the app is valid on disk. `codesign -d` should list three authorities: your Apple Development certificate, Apple Worldwide Developer Relations, and Apple Root CA. `Authority=(unavailable)` means the signature is not usable. Install error `0xe8008018` (`The identity used to sign the executable is no longer valid`) means the certificate that signed the app is not the certificate inside the profile, even when both names look the same. Sign again with the `.p12` that matches the profile.

### 3. Install on every phone

Copy `GADSFileTransfer.ipa` to each computer that has phones plugged in. Do not sign again on those computers.

`ios` is [go-ios](https://github.com/danielpaulus/go-ios). Install uses lockdown and does not need `ios tunnel start`. A tunnel warning during install can be ignored.

```sh
ios install --path=GADSFileTransfer.ipa --udid=PHONE_UDID
```

Repeat for each UDID from `ios list` on that computer. Then open **GADS File Transfer** on the phone and tap **Allow** for Photos. That tap is once per phone.

A phone whose UDID is missing from the provisioning profile is rejected. Add it to the profile, sign once more, and install that new IPA.

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

## License

Copyright (C) 2026 TheDevWhoSaysNi.

GNU Affero General Public License v3.0, the same license as [GADS](https://github.com/shamanec/GADS). See [LICENSE](LICENSE).
