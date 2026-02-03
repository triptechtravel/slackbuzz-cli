---
title: "slackbuzz completion"
description: "Auto-generated reference for slackbuzz completion"
---

## slackbuzz completion

Generate shell completion scripts

### Synopsis

Generate shell completion scripts for slackbuzz.

To load completions:

Bash:
  $ source <(slackbuzz completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ slackbuzz completion bash > /etc/bash_completion.d/slackbuzz
  # macOS:
  $ slackbuzz completion bash > $(brew --prefix)/etc/bash_completion.d/slackbuzz

Zsh:
  $ source <(slackbuzz completion zsh)
  # To load completions for each session, execute once:
  $ slackbuzz completion zsh > "${fpath[1]}/_slackbuzz"

Fish:
  $ slackbuzz completion fish | source
  # To load completions for each session, execute once:
  $ slackbuzz completion fish > ~/.config/fish/completions/slackbuzz.fish

PowerShell:
  PS> slackbuzz completion powershell | Out-String | Invoke-Expression


```
slackbuzz completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [slackbuzz](/slackbuzz-cli/reference/slackbuzz/)	 - Slack CLI - message, search, and manage channels from the command line

