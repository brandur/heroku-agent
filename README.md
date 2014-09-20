# heroku-agent

heroku-agent is a lightweight process that listens on a Unix domain socket and can communicate with the [Heroku CLI](https://github.com/heroku/heroku) and [hk](https://github.com/heroku/hk) to provide more expendient fulfillment of API requests and better convenience of use.

It provides the following features:

* **Conditional requests:** Caches response bodies and checks their freshness via etag, which can greatly reduce the amount of data that needs to be sent over the wire.
* **TCP connection pooling:** heroku-agent can keep connections open to the Heroku API and its various peripheral services, which avoids the expensive overhead of opening SSL connections for requests that occur during the keep-alive window.
* **Second factor management:** Stores and manages the lifecycle of a second authentication factor so that clients are only re-prompted when necessary.

## Benchmarks

### hk

``` bash
# WITH heroku-agent
$ time (for i in `seq 1 100`; do hk apps > /dev/null ; done)

real    20.04s
user    1.77s
sys     1.21s

# WITHOUT heroku-agent
$ time (for i in `seq 1 100`; do hk apps > /dev/null ; done)

real    88.06s
user    12.64s
sys     2.09s
```
