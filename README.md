# Haiku CLI

# Building
All of the build scripts are encapsulated within the [Makefile](./Makefile)

To build the project in a standard way for your OS, simply run:
```
make build
```
This will output to `bin/haiku`

To build under multiple targets and architectures:
```
make build-release
```
This is made to be run under CI, but if running on a Mac, `sha256sum` must be installed.
This can be done with `brew install coreutils`.

# Releasing
To release a new version of the CLI for consumption, a new git tag must be created.

```
$ ./hack/tag.sh v1.0.0
```
This will create a git tag and push it to the origin.
This will kick off a Github Action, which will do the following:
1. Build the project across multiple targets (linux, mac) and multiple architectures (64bit, ARM)
2. Create a new Github Release based off of the new tag and attach the binaries to the release.

# Installing Haiku CLI

## Download from releases page
1. Go to the [Releases Page](https://github.com/haikuapp/cli/releases).
2. Download the binary for your operating system.
3. Rename the downloaded file to `haiku`.
4. Add execute permissions to the binary. E.g., on linux and mac: `chmod u+x haiku`.
5. Put the binary somewhere on your `PATH`. E.g., on linux and mac: `mv haiku /usr/local/bin/haiku`.

## Install via a package manager
TODO
