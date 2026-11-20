---
title: "slackbuzz file delete"
description: "Auto-generated reference for slackbuzz file delete"
---

## slackbuzz file delete

Delete files

### Synopsis

Delete files by ID (F…). Deleting a message leaves its files in Slack;
this removes them. The IDs are in send --file and file upload --json output.

Prompts for confirmation in interactive mode; --confirm is required otherwise.

```
slackbuzz file delete <file-id>... [flags]
```

### Examples

```
  slackbuzz file delete F0CK25W3ABH F0CKKDV2K8R --confirm
```

### Options

```
      --as-bot    Delete as the bot (for files the bot posted)
      --confirm   Skip confirmation prompt
  -h, --help      help for delete
```

### SEE ALSO

* [slackbuzz file](/slackbuzz-cli/reference/slackbuzz_file/)	 - Search, upload, and manage files

