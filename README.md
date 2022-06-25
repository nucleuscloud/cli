# Nucleus CLI

## Dependencies
golangci-lint:
```brew install golangci-lint```

Command Line Tools Package for XCode:
```xcode-select --install```

## Enable private repo access (using ssh)
```export GOPRIVATE=github.com/nucleuscloud```

Add the following to ~/.gitconfig
```
[url "git@github.com:nucleuscloud"]
    insteadOf = https://github.com/nucleuscloud
```
## Building
All of the build scripts are encapsulated within the [Makefile](./Makefile)

To build the project in a standard way for your OS, simply run:
```
make build
```
This will output to `bin/nucleus`

To build under multiple targets and architectures:
```
make build-release
```
This is made to be run under CI, but if running on a Mac, `sha256sum` must be installed.
This can be done with `brew install coreutils`.

## Releasing
To release a new version of the CLI for consumption, a new git tag must be created.

```
$ ./hack/tag.sh v1.0.0
```
This will create a git tag and push it to the origin.
This will kick off a Github Action, which will do the following:
1. Build the project across multiple targets (linux, mac) and multiple architectures (64bit, ARM)
2. Create a new Github Release based off of the new tag and attach the binaries to the release.

## Installing Nucleus CLI

### Homebrew
You can install Nucleus CLI directly from Homebrew
```sh
brew install nucleuscloud/tap/nucleus
```

### Download from releases page
1. Go to the [Releases Page](https://github.com/nucleuscloud/cli/releases).
2. Download the tarball for your operating system: `tar xzf <path-to-tar.gz> nucleus`
5. Put the binary somewhere on your `PATH`. E.g., on linux and mac: `mv nucleus /usr/local/bin/nucleus`.

### Install via a package manager
Ensure you have `go` installed.
This is only possible today for Nucleus devs as the CLI depends on types that live in the private organization
```sh
make
./bin/nucleus
```
