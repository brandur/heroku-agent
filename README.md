# heroku-agent

heroku-agent is a lightweight process that listens on a Unix domain socket and can communicate with the [Heroku CLI](https://github.com/heroku/heroku) and [hk](https://github.com/heroku/hk) to provide more expendient fulfillment of API requests and better convenience of use.

It provides the following features:

* **Conditional requests:** Caches response bodies and checks their freshness via etag, which can greatly reduce the amount of data that needs to be sent over the wire.
* **TCP connection pooling:** heroku-agent can keep connections open to the Heroku API and its various peripheral services, which avoids the expensive overhead of opening SSL connections for requests that occur during the keep-alive window.
* **Second factor management:** Stores and manages the lifecycle of a second authentication factor so that clients are only re-prompted when absolutely necessary.
