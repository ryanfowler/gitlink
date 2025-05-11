# gitlink

Get a permanent link to a file location in a git repository.

## Install

```sh
go install github.com/ryanfowler/gitlink@latest
```

## Usage

```
Usage: gitlink [OPTIONS] <FILEPATH> <LINE_NUM>

Arguments:
  <FILEPATH>  Path to the file
  <LINE_NUM>  Line number to link to

Options:
  --blame  Link to the git blame view
  --open   Open link in the default browser
```
