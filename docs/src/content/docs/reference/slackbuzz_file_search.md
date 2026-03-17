---
title: "slackbuzz file search"
description: "Auto-generated reference for slackbuzz file search"
---

## slackbuzz file search

Search files (requires user token)

### Synopsis

Search files shared in Slack using the search.files API.

This command requires a user token (xoxp-) with search:read scope.

```
slackbuzz file search <query> [flags]
```

### Examples

```
  # Search for files
  slackbuzz file search "design spec"

  # Filter by file type
  slackbuzz file search "report" --type pdf

  # Filter by channel
  slackbuzz file search "report" --channel #engineering

  # Filter by uploader
  slackbuzz file search "diagram" --from @alice

  # Pagination
  slackbuzz file search "report" --page 2
```

### Options

```
      --channel string    Filter by channel
      --from string       Filter by uploader
  -h, --help              help for search
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of results per page (default 20)
      --page int          Page number for pagination (default 1)
      --template string   Format JSON output using a Go template
      --type string       Filter by file type (e.g. pdf, png, zip)
```

### SEE ALSO

* [slackbuzz file](/slackbuzz-cli/reference/slackbuzz_file/)	 - Search, upload, and manage files

