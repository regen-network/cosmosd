# Cosmos Upgrade Manager

This is a tiny little shim around Cosmos SDK binaries that use the upgrade
module that allows for smooth and configurable management of upgrading
binaries as a live chain is upgraded, and can be used to simplify validator
devops while doing upgrades or to make syncing a full node for genesis
simple.

A few very simple conventions are used to make this process as painless as
possible:

* set `DAEMON_HOME` to the location where upgrade binaries should be kept (can
be `$HOME/.gaiad` or `$HOME/.xrnd`)
* we use [go-getter](https://github.com/hashicorp/go-getter) for all URI's which
includes the ability to automatically compute SHA256 checksum's and unpack archives
* place the genesis binary at `$DAEMON_HOME/upgrade_manager/genesis` or point
`GENESIS_BINARY` to a [go-getter](https://github.com/hashicorp/go-getter) URI
to retrieve it from
* place the binary for each upgrade at `$DAEMON_HOME/upgrade_manager/<name>`
where `<name>` is the URI-encoded name of the upgrade as specified in the upgrade
module plan
* or, store an os/architecture -> binary URI map in the upgrade plan info field
as JSON or YAML under the `"binaries"` key, eg:
```json
{
  "binaries": {
    "linux/amd64":"https://example.com/gaiad?checksum=sha256:b7d96c89d09d9e204f5fedc4d5d55b21"
  }
}
```
* or, set the upgrade plan to URI that points to a YAML or JSON file with the above structure 
that can be retrieved by [go-getter](https://github.com/hashicorp/go-getter) 
* all arguments passed to the upgrade manager command will be passed to the
current daemon binary
* the upgrade manager will monitor the stdout of the daemon to look for signals
from the upgrade module indicating a pending or required upgrade and act
appropriately, preferring on-disk binaries when they are present and defaulting
to binaries provided in the on-chain upgrade plan when they are not
