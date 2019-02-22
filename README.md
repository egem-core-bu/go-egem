## Go EGEM

Official golang implementation of the EGEM protocol.

Automated builds are available for stable releases and the unstable master branch.
Binary archives are published at https://git.egem.io/team/egem-binaries.

## Building the source

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum)
on the wiki.

Building egem requires both a Go (version 1.11 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make egem

or, to build the full suite of utilities:

    make all

## Executables

The go-egem project comes with several wrappers/executables found in the `cmd` directory.

| Command    | Description |
|:----------:|-------------|
| **`egem`** | Our main EGEM CLI client. It is the entry point into the EGEM network. |
| **`stats`** | Quarrynode JSON output for usage with our system. |


## Running egem

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://git.egem.io/team/go-egem/wikis/Command-Line-Options)), but we've
enumerated a few common parameter combos to get you up to speed quickly on how you can run your
own EGEM instance.

## Custom Commands
  * `--quarrynode` Enable the Quarrynode server.
  * `--database-handles` Allows more handles when fast syncing, speeds up chain sync on fresh start.
  * `--atxi --atxi.autobuild` Allows indexing of transactions and to make use of some new commands.

### Full node on the main EGEM network

By far the most common scenario is people wanting to simply interact with the Ethereum network:
create accounts; transfer funds; deploy and interact with contracts. For this particular use-case
the user doesn't care about years-old historical data, so we can fast-sync quickly to the current
state of the network. To do so:

```
$ egem --syncmode fast --database-handles 100000 console
```

This command will:

 * Start egem in fast sync mode (default, can be changed with the `--syncmode` flag), causing it to
   download more data in exchange for avoiding processing the entire history of the EGEM network,
   which is very CPU intensive.

### Quarrynode on the main EGEM network

This will start egem in quarrynode mode enabling web3,eth,net:

```
$ egem --syncmode fast --quarrynode console
```

This command will:

  * Start egem in fast sync mode and quarrynode mode at the same time, with `--rpcaddr=0.0.0.0` `--rpccorsdomain=*` `--rpcvhost=*` and does not allow an account to be unlocked with the --quarrynode flag enabled.

### Configuration

As an alternative to passing the numerous flags to the `egem` binary, you can also pass a configuration file via:

```
$ egem --config /path/to/your_config.toml
```

To get an idea how the file should look like you can use the `dumpconfig` subcommand to export your existing configuration:

```
$ egem --your-favourite-flags dumpconfig
```

Do not forget `--rpcaddr 0.0.0.0`, if you want to access RPC from other containers and/or hosts. By default, `egem` binds to the local interface and RPC endpoints is not accessible from the outside.

### Programatically interfacing Geth nodes

As a developer, sooner rather than later you'll want to start interacting with EGEM and the EtherGem
network via your own programs and not manually through the console. To aid this, EGEM has built in
support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereum/wiki/wiki/JSON-RPC) and
[Geth specific APIs](https://git.egem.io/team/go-egem/wiki/Management-APIs)). These can be
exposed via HTTP, WebSockets and IPC (unix sockets on unix based platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by EGEM, whereas the HTTP
and WS interfaces need to manually be enabled and only expose a subset of APIs due to security reasons.
These can be turned on/off and configured as you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpcport` HTTP-RPC server listening port (default: 8895)
  * `--rpcapi` API's offered over the HTTP-RPC interface (default: "eth,net,web3")
  * `--rpccorsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface (default: "localhost")
  * `--wsport` WS-RPC server listening port (default: 8896)
  * `--wsapi` API's offered over the WS-RPC interface (default: "eth,net,web3")
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to connect
via HTTP, WS or IPC to a EGEM node configured with the above flags and you'll need to speak [JSON-RPC](http://www.jsonrpc.org/specification)
on all transports. You can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based transport before
doing so! Hackers on the internet are actively trying to subvert Ethereum nodes with exposed APIs!
Further, all browser tabs can access locally running webservers, so malicious webpages could try to
subvert locally available APIs!**

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
