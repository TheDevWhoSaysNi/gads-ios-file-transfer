package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/TheDevWhoSaysNi/gads-ios-file-transfer/internal/transfer"
	"github.com/TheDevWhoSaysNi/gads-ios-file-transfer/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		os.Exit(serve(os.Args[2:]))
	case "devices":
		os.Exit(devices())
	case "push":
		os.Exit(push(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `GADS File Transfer copies a file from this computer onto an iPhone.

  gads-file-transfer devices
  gads-file-transfer push --udid UDID [--provider URL] file
  gads-file-transfer serve [--listen 127.0.0.1:8787] [--token SECRET] [--provider URL]

Photos and videos land in Photos and in Files. Other files land in Files.
The GADS File Transfer app must be installed on the phone.
`)
}

func devices() int {
	list, err := transfer.ListDevices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(list)
	return 0
}

func push(args []string) int {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	udid := fs.String("udid", "", "phone UDID")
	provider := fs.String("provider", "", "GADS provider base URL, for example http://127.0.0.1:10001")
	_ = fs.Parse(args)
	if *udid == "" || fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gads-file-transfer push --udid UDID file")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	launch := transfer.LaunchWithInstruments
	if *provider != "" {
		launch = transfer.ProviderLaunch(*provider)
	}
	outcome, err := transfer.Send(ctx, *udid, fs.Arg(0), fs.Arg(0), launch)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *provider != "" {
		transfer.ProviderHome(*provider, *udid)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(outcome)
	return 0
}

func serve(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:8787", "address to serve the upload page")
	token := fs.String("token", os.Getenv("GADS_TRANSFER_TOKEN"), "required when listen is not localhost, or when a tunnel reaches this page")
	provider := fs.String("provider", os.Getenv("GADS_PROVIDER_URL"), "GADS provider base URL on this same computer")
	_ = fs.Parse(args)
	if web.PublicListen(*listen) && *token == "" {
		fmt.Fprintln(os.Stderr, "refusing to listen on a public address without --token")
		return 2
	}
	srv := &web.Server{Token: *token, Provider: *provider}
	fmt.Printf("GADS File Transfer at http://%s\n", *listen)
	if err := web.ListenAndServe(*listen, srv.Handler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
