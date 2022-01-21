Haiku CLI

# Releasing
To release a new version of the CLI for consumption, a new git tag must be created.

```
$ ./hack/tag.sh v1.0.0
```

This will create a git tag and push it to the origin.
Upon this push, a github action will start, which will build the project across multiple targets, and upload them to a github release.

# Installing Haiku CLI

## Download from releases page
1. Go to the [Releases Page](https://github.com/haikuapp/cli/releases).
2. Download the binary for your operating system.
3. rename the downloaded file to `haiku`.
4. Add execute permissions to the binary. E.g., on linux and mac: `chmod u+x haiku`.
5. Put the binary somewhere on your `PATH`. E.g., on linux and mac: `mv haiku /usr/local/bin/haiku`.

## Install via a package manager
TODO
