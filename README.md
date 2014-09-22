# heroku-agent

heroku-agent is a lightweight process that can communicate with the [Heroku CLI](https://github.com/heroku/heroku) and [hk](https://github.com/heroku/hk) to provide more expendient fulfillment of API requests and better convenience of use.

It provides the following features:

* **Conditional requests:** Caches response bodies and checks their freshness via etag, which can greatly reduce the amount of data that needs to be sent over the wire.
* **TCP connection pooling:** heroku-agent can keep connections open to the Heroku API and its peripheral services, which avoids the expensive overhead of opening SSL connections for requests that occur during the keep-alive window.
* **Second factor management:** Stores and manages the lifecycle of a second authentication factor so that clients are only re-prompted when necessary.

## Installation

# Common

``` bash
$ go get -u github.com/brandur/heroku-agent

# in your .bashrc or .zshrc, add this:
export HEROKU_AGENT_SOCK=~/.heroku-agent.sock
(nohup heroku-agent -v >> ~/.heroku-agent.log &)
```

## hk

``` bash
$ hk update
# OR (if installed manually via Go)
$ go get -u github.com/heroku/hk
$ hk apps
```

## heroku

**WARNING:** heroku-agent requires a yet-as-unreleased version of the Heroku CLI to function. The following instructions will **not work quite yet**. Ask Brandur for details on how to install a prerelease.

``` bash
$ heroku update
$ heroku plugins:update
$ heroku plugins:install https://github.com/brandur/heroku-agent-plugin
$ heroku config
```

## Benchmarks

### heroku

We observe roughly a 2.5x speedup with the Heroku CLI. This improvement might be greater, but the CLI spends a lot of time just warming up. See hk below for a much more dramatic increase.

```
# WITH heroku-agent
$ time (for i in `seq 1 100`; do heroku info -a mutelight > /dev/null ; done)

real    111.28s
user    44.02s
sys     5.64s

# WITHOUT heroku-agent
$ time (for i in `seq 1 100`; do heroku info -a mutelight > /dev/null ; done)

real    277.81s
user    50.81s
sys     6.34s
```

### hk

More than a 4x speed improvement.

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
