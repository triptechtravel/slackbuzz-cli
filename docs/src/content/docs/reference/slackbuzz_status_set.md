---
title: "slackbuzz status set"
description: "Auto-generated reference for slackbuzz status set"
---

## slackbuzz status set

Set your Slack status

### Synopsis

Set your Slack status text and emoji.

The emoji argument is optional and should be in :emoji: format.
Use --until to set an expiration (e.g. 2h, 1d).

```
slackbuzz status set <text> [emoji] [flags]
```

### Examples

```
  slackbuzz status set "In a meeting" :calendar:
  slackbuzz status set "Coding" :computer: --until 2h
  slackbuzz status set "Lunch" :fork_and_knife: --until 1h
```

### Options

```
  -h, --help           help for set
      --until string   Status expiration (e.g. 2h, 1d)
```

### SEE ALSO

* [slackbuzz status](/slackbuzz-cli/reference/slackbuzz_status/)	 - Slack status management

