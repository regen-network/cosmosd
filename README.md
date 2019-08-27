# Cosmos Upgrade Manager

This is a tiny little shim around Cosmos SDK binaries that use the upgrade
module that allows for smooth and configurable management of upgrading
binaries as a live chain is upgraded, and can be used to simplify validator
devops while doing upgrades or to make syncing a full node for genesis
simple. The upgrade manager will monitor the stdout of the daemon to look 
for messages from the upgrade module indicating a pending or required upgrade 
and act appropriately. (With better integrations possible in the future)

## Arguments

`upgrader` is a shim around a native binary. All arguments passed to the upgrade manager 
command will be passed to the current daemon binary (as a subprocess).
 It will return stdout and stderr of the subprocess as
it's own. Because of that, it cannot accept any command line arguments, nor
print anything to output (unless it dies before executing a binary).

Configuration will be passed in the followingenvironmental variables:

* `DAEMON_HOME` is the location where upgrade binaries should be kept (can
be `$HOME/.gaiad` or `$HOME/.xrnd`)
* `DAEMON_NAME` is the name of the binary itself (eg. `xrnd`, `gaiad`)
* `DAEMON_DOWNLOAD` (optional) if set to `on` will enable auto-downloading of new binaries
(for security reasons, this is intended for fullnodes rather than validators)

## Folder Layout

`$DAEMON_HOME/upgrade_manager` is expected to belong completely to the upgrade manager and subprocesses
constrolled by it. Under this folder, we will see the following:

```
- bin
  - upgrade_manager (this binary)
- genesis
  - bin
    - $DAEMON_NAME
- upgrades
  - <name>
    - bin
      - $DAEMON_NAME
- current -> upgrades/foo
```

Each version of the chain is stored under either `genesis` or `upgrades/<name>`, which holds `bin/$DAEMON_NAME`
along with any other needed files (maybe the cli client? maybe some dlls?). `current` is a symlink to the currently
active folder (so `current/bin/$DAEMON_NAME` is the binary)

Note: the `<name>` after `upgrades` is the URI-encoded name of the upgrade as specified in the upgrade module plan.

**Question** where is the state of the blockchain (config, data) stored? Under the update manager, or elsewhere?

## Usage

Basic Usage:

* The admin is responsible for installing the `upgrade_manager` and setting it as a eg. systemd service to auto-restart
* The admin is responsible for installing the `genesis` folder manually
* The admin is responsible for setting the `current` link to point to genesis
* The admin is (generally) responsible for installing the `upgrades/<name>` folders manually
* The upgrade manager handles switching over the binaries at the correct points, so the admin can prepare days in advance and relax at upgrade time

Note that chains that wish to support upgrades may package up a genesis upgrade manager tar file with this info, just as they
prepare the genesis binary tar file. In fact, they may offer a tar file will all upgrades up to current point for easy download
for those who wish to sync a fullnode from start.

The `DAEMON` specific code, like the tendermint config, the application db, syncing blocks, etc is done as normal.
The same eg. `GAIA_HOME` directives and command-line flags work, just the binary name is different.

## Upgradeable Binary Specification

In the basic version, the upgrade_manager will read the stdout log messages
to determine when an upgrade is needed. We are considering more complex solutions
via signaling of some sort, but starting with the simple design:

* when an upgrade is needed the binary will print a line that matches this
regular expression: `UPGRADE "(.*)" NEEDED at height (\d+):(.*)`.
* the second match in the above regular expression can be a JSON object with
a `binaries` key as described above

The name (first regexp) will be used to select the new binary to run. If it is present,
the current subprocess will be killed, `current` will be upgraded to the new directory, 
and the new binary will be launched.

**Question** should we just kill the upgrade manager after it does the updates?
so it gets a clean restart and just runs the new binary (under `current`).
it should be safe to restart (as a service).

## Auto-Download

Generally, the system requires that the administrator place all relevant binaries
on the disk before the upgrade happens. However, for people who don't need such
control and want an easier setup (maybe they are syncing a non-validating fullnode
and want to  do little maintenance), there is another option.

If you set `DAEMON_DOWNLOAD=on` then when an upgrade is triggered and no local binary
can be found, the upgrade_manager will attempt to download and install the binary itself.
The plan stored in the upgrade module has an info field for arbitrary json.
This info is expected to be outputed on the halt log message. There are two
valid format to specify a download in such a message:

1. Store an os/architecture -> binary URI map in the upgrade plan info field
as JSON (or YAML??) under the `"binaries"` key, eg:
```json
{
  "binaries": {
    "linux/amd64":"https://example.com/gaiad?checksum=sha256:b7d96c89d09d9e204f5fedc4d5d55b21"
  }
}
```
2. Store a link to a file that contains all information in the above format (eg. if you want
to specify lots of binaries, changelog info, etc without filling up the blockchain).
```json
{
  "link":"https://example.com/testnet-1001-info.json"
}
```
This file contained in link will be retrieved by [go-getter](https://github.com/hashicorp/go-getter) 
and the "binaries" field will be parsed as above.

If there is no local binary, `DAEMON_DOWNLOAD=on`, and we can access a canonical url for the new binary,
then the upgrade_manager will download it with [go-getter](https://github.com/hashicorp/go-getter) and
unpack it into the `upgrades/<name>` folder to be run as if we installed it manually
