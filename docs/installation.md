---
title: Installation
layout: default
nav_order: 1
---

# Installation

You can install *dnspyre* using [Homebrew](https://brew.sh) package manager, see [Homebrew Formula](https://formulae.brew.sh/formula/dnspyre)

```
brew install dnspyre
```

Also you can use standard [Go tooling](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies) to install *dnspyre*

```
go install github.com/tantalor93/dnspyre/v3@latest
```

Or you can download latest *dnspyre* binary archive for your operating system and architecture [here](https://github.com/Tantalor93/dnspyre/releases/latest)

## Bash/ZSH Shell completion
When *dnspyre* is installed using [Homebrew](https://brew.sh), the shell completions are installed automatically, if Homebrew is configured to [install them](https://docs.brew.sh/Shell-Completion)

Otherwise you have to setup completions manually:

For **ZSH**, add to your `~/.zprofile` (or equivalent ZSH configuration file)

```
eval "$(dnspyre --completion-script-zsh)"
```

For **Bash**, add to your `~/.bash_profile` (or equivalent Bash configuration file)

```
eval "$(dnspyre --completion-script-bash)"
```

# Docker image
if you don't want to install *dnspyre* locally, you can use prepared [Docker image](https://github.com/Tantalor93/dnspyre/pkgs/container/dnspyre),
for example 

```
docker run ghcr.io/tantalor93/dnspyre --server 8.8.8.8 google.com
```
