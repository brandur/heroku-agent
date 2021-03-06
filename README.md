# heroku-agent

heroku-agent is a lightweight process that can communicate with the [Heroku CLI](https://github.com/heroku/heroku) and [hk](https://github.com/heroku/hk) to provide more expendient fulfillment of API requests and better convenience of use.

It provides the following features:

* **Conditional requests:** Caches response bodies and checks their freshness via etag, which can greatly reduce the amount of data that needs to be sent over the wire.
* **TCP connection pooling:** heroku-agent can keep connections open to the Heroku API and its peripheral services, which avoids the expensive overhead of opening SSL connections for requests that occur during the keep-alive window.
* **Second factor management:** Stores and manages the lifecycle of a second authentication factor so that clients are only re-prompted when necessary.

## Installation

### Common (do this for both hk and Heroku CLI)

``` bash
# note that Go 1.3+ is required
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

``` bash
$ heroku update
$ heroku plugins:update
$ heroku plugins:install https://github.com/brandur/heroku-agent-plugin
$ heroku apps

# verify that the request happened against heroku-agent
$ tail ~/.heroku-agent.log
```

### HTTP Git

A credential helper is bundled which allows Git to use a privileged token procured by heroku-agent (if one is available) to push to paranoid apps over HTTP Git without requiring pre-authentication.

Make sure that heroku-agent is up-to-date:

```
go get -u github.com/brandur/heroku-agent
```

Make sure that your version of Git supports credential helpers. You're looking for something that's not ancient (2+):

```
git --version
```

Now install the credential helpers:

```
curl https://raw.githubusercontent.com/git/git/master/contrib/credential/netrc/git-credential-netrc > ~/bin/git-credential-netrc
chmod +x ~/bin/git-credential-netrc

curl https://raw.githubusercontent.com/brandur/heroku-agent/master/contrib/git-credential-heroku-agent > ~/bin/git-credential-heroku-agent
chmod +x ~/bin/git-credential-heroku-agent
```

Then add the helper to your `~/.gitconfig`:

```
[credential]
	helper = heroku-agent
```

Now you have heroku-agent procure a privileged token and deploy to paranoid apps normally:

```
# if necessary, convert your remote from Git to HTTP
heroku git:remote --http -r heroku

# authorizing is only necessary if heroku-agent isn't already holding a token
hk authorize

git push heroku master
```

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
