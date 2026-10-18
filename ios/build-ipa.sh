#!/bin/sh
# Build an unsigned IPA on a Mac with full Xcode. Sign it afterwards.
# The certificate inside the .p12 must be one of the certificates inside
# the provisioning profile. See the README.
set -eu
cd "$(dirname "$0")"
BUILD="$(pwd)/build"
rm -rf "$BUILD"
xcodebuild \
  -project GADSFileTransfer.xcodeproj \
  -target GADSFileTransfer \
  -configuration Release \
  -sdk iphoneos \
  CODE_SIGNING_ALLOWED=NO \
  build \
  SYMROOT="$BUILD" \
  OBJROOT="$BUILD"
APP="$(find "$BUILD" -name 'GADSFileTransfer.app' -type d | head -n 1)"
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
