#!/bin/sh
# Build an unsigned IPA on a Mac. Sign it afterwards with the same zsign
# identity you use for WebDriverAgent, then install with `ios install`.
set -eu
cd "$(dirname "$0")"
DERIVED="$(pwd)/build"
rm -rf "$DERIVED"
xcodebuild \
  -project GADSFileTransfer.xcodeproj \
  -target GADSFileTransfer \
  -configuration Release \
  -sdk iphoneos \
  -derivedDataPath "$DERIVED" \
  CODE_SIGNING_ALLOWED=NO \
  build
APP="$(find "$DERIVED" -name 'GADSFileTransfer.app' -type d | head -n 1)"
if [ -z "$APP" ]; then
  echo "Build finished but GADSFileTransfer.app was not found." >&2
  exit 1
fi
STAGE="$(mktemp -d)"
mkdir -p "$STAGE/Payload"
cp -R "$APP" "$STAGE/Payload/"
IPA="$(pwd)/GADSFileTransfer-unsigned.ipa"
rm -f "$IPA"
(cd "$STAGE" && zip -qry "$IPA" Payload)
rm -rf "$STAGE"
echo "Wrote $IPA"
echo "Sign it, then: ios install --path=the-signed.ipa"
