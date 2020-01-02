# yq

This is an implementation of https://github.com/kislyuk/yq in go. It is mostly
a drop in replacement for https://github.com/kislyuk/yq for YAML consumption
from the command-line, this does not implement the XML parsing functionality.

## What does not work?

- The YAML/JSON is sorted after being manipulated by yq, this is due to the
behavior of `encoding/json` which sorts keys by default.
- command line flags cannot be combined e.g:

```
yq -sc . foo.yaml
```

does not work, instead you have to do:

```
yq -s -c . foo.yaml
```

This is a limitation of the golang flag stdlib, this will probably be replaced
in the future.


## FAQ

- why re-implement https://github.com/kislyuk/yq?
I have need for a smaller tool for parsing yaml from the cli,
https://github.com/kislyuk/yq includes requires Python and it's toolchain to
install it which introduces unwanted dependencies.
- why not https://github.com/mikefarah/yq? It is not compatible and feature
rich as jq, I am familiar with jq and would like to not have to learn a
similar but different tool.
