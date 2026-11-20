---
title: "slackbuzz file upload"
description: "Auto-generated reference for slackbuzz file upload"
---

## slackbuzz file upload

Upload files to a channel or DM

### Synopsis

Upload one or more files to a Slack channel or direct message.

The last argument is the channel or user target. All preceding arguments are
file paths to upload. All the files are shared together as ONE message, with
--comment as its text (formatted like message send: Markdown to mrkdwn and
@mentions; --raw keeps it as typed).

To post a message with files attached, message send --file does the same.

The channel argument accepts #channel-name, channel ID, @username, or user ID.

```
slackbuzz file upload <file-path>... <channel|user> [flags]
```

### Examples

```
  # Upload a single file to a channel
  slackbuzz file upload report.pdf #general

  # Upload multiple files
  slackbuzz file upload chart1.png chart2.png chart3.png #analytics

  # Upload with title and comment
  slackbuzz file upload data.csv #sales --title "Q4 Revenue" --comment "Updated data"

  # Upload to a DM
  slackbuzz file upload notes.txt @alice

  # Upload to a thread
  slackbuzz file upload results.png #dev --thread-ts 1706000000.000000
```

### Options

```
      --comment string     Text of the message the file(s) are posted in
  -h, --help               help for upload
      --jq string          Filter JSON output using a jq expression
      --json               Output JSON
      --raw                Disable Markdown→mrkdwn conversion of --comment
      --template string    Format JSON output using a Go template
      --thread-ts string   Thread timestamp to upload into
      --title string       File title in Slack (applies to first file)
```

### SEE ALSO

* [slackbuzz file](/slackbuzz-cli/reference/slackbuzz_file/)	 - Search, upload, and manage files

