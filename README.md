# heroku-agent

heroku-agent is a lightweight process that can communicate with the [Heroku CLI](https://github.com/heroku/heroku) and [hk](https://github.com/heroku/hk) to provide more expendient fulfillment of API requests and better convenience of use.

It provides the following features:

* **Conditional requests:** Caches response bodies and checks their freshness via etag, which can greatly reduce the amount of data that needs to be sent over the wire.
* **TCP connection pooling:** heroku-agent can keep connections open to the Heroku API and its peripheral services, which avoids the expensive overhead of opening SSL connections for requests that occur during the keep-alive window.
* **Second factor management:** Stores and manages the lifecycle of a second authentication factor so that clients are only re-prompted when necessary.

## Installation

### Common (do this for both hk and Heroku CLI)

``` bash
$ go get -u github.com/brandur/heroku-agent

# in your .bashrc or .zshrc, add this:
export HEROKU_AGENT_SOCK=~/.heroku-agent.sock
(nohup heroku-agent -v >> ~/.heroku-agent.log &)
```

Alternatively, if you are on OSX, you can [try this .plist file](https://gist.github.com/dpiddy/9130e67ec24862706516).

### hk

``` bash
$ hk update
# OR (if installed manually via Go)
$ go get -u github.com/heroku/hk
$ hk apps
```

### heroku

**WARNING:** heroku-agent requires a yet-as-unreleased version of the Heroku CLI to function. Until then, these instructions are a bit janky (and somewhat fragile).

``` bash
$ heroku update
$ heroku plugins:update
$ heroku plugins:install https://github.com/brandur/heroku-agent-plugin
$ heroku apps

# verify that the request happened against heroku-agent
$ tail ~/.heroku-agent.log
```

If you have been using heroku-agent successfully, but receive an error that looks like this:

```
$ heroku config -a core-db

 !    Heroku client internal error.
 !    Search for help at: https://help.heroku.com
 !    Or report a bug at: https://github.com/heroku/heroku/issues/new

    Error:       Proxy is invalid (Excon::Errors::ProxyParseError)
...
```

The likely cause is that your Toolbelt has updated itself and overwritten the custom Excon gem. The fix is to follow the installation instructions again.

## Benchmarks

### hk

hk regularly gets more than a 4x improvement in speed for commands that are performed relatively close together.

``` bash
# WITH heroku-agent
$ time (for i in `seq 1 100`; do hk info -a mutelight > /dev/null ; done)

real    21.66s
user    0.94s
sys     1.46s

# WITHOUT heroku-agent
$ time (for i in `seq 1 100`; do hk info -a mutelight > /dev/null ; done)

real    94.21s
user    22.99s
sys     2.20s
```

### heroku

Similarly, we observe roughly a 2x speedup with the Heroku CLI. This improvement might be greater, but the CLI spends a lot of time warming up to execute just a single command.

```
# WITH heroku-agent
$ time (for i in `seq 1 100`; do heroku info -a mutelight > /dev/null ; done)

real    90.32s
user    42.07s
sys     5.31s

# WITHOUT heroku-agent
$ time (for i in `seq 1 100`; do heroku info -a mutelight > /dev/null ; done)

real    176.66s
user    52.06s
sys     6.47s
```
