---
title: Installation
layout: default
nav_order: 1
---

# Installation

using `brew`
```
brew tap tantalor93/dnspyre
brew install dnspyre
```

or `go install`
```
go install github.com/tantalor93/dnspyre/v2@latest
```

## Bash/ZSH Shell completion
For **ZSH**, add to your `~/.zprofile` (or equivalent ZSH configuration file)
```
eval "$(dnspyre --completion-script-zsh)"
```

For **Bash**, add to your `~/.bash_profile` (or equivalent Bash configuration file)
```
eval "$(dnspyre --completion-script-bash)"
```

# Docker image
if you don't want to install `dnspyre` locally, you can use prepared [Docker image](https://hub.docker.com/r/tantalor93/dnspyre/tags),
for example 
```
docker run tantalor93/dnspyre --server 8.8.8.8 google.com
```
