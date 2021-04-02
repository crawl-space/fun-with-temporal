# Temporal Fun

This repo shows you some fun with temporal, running a workflow under
temporal, but having the bulk of the work done by an external REST API
that calls back when work is done.

## The setup

- You have an external system calling your API (say, a webhook that fires when new PRs are open). *This is handled by the go worker's `pr` http handler.
- Once the API is called, it enqueues a workflow run to deal with PRs. This is put in termporal.
- worker process picks up this workflow run, and calls out to a leaf process that does the work. *this is the node.js process in the leaf dir.
- The leaf worker gets a callback url. When it's done (after 10 seconds) it calls back to complete the work.
- The worker waits for two such jobs running in parallel, then does some extra work internally, then does some further external work (say, to set a commit status on the original PR). Then its done!

Temporal handles:
- queueing and limiting the work after accepting that original `/pr` call
- checkpointing steps in the workflow (in `workflow.go`) so that in the event of failure, work is not lost.
- retries for any part of originally sending the work to the leaf, or if it does not call back (or heartbeat) before a dealine.

## Run it

- Run temporal via docker-compose as described here: https://github.com/temporalio/docker-compose#how-to-use
- Install Go
- Install Node
- run `make` in the root of this dir to build the worker/server process.
- `cd` into `leaf` and run `npm i` to get all them node deps.
- in one terminal, start the leaf worker in the `leaf` dir: `node index.js`
- in another terminal, start the worker/server from the root dir: `./worker`

Now you're ready to enqueue work.
- install [httpie](https://httpie.io/) because it's really good and you deserve nice things. you do. it's ok to indulge once in a while in a nice CLI tool.
- Set up some work! run `http POST 127.0.0.1:6007/pr repo=jbowes/repl pr=123 old=bahbah new=naynay` To simulate a PR against the `jbowes/repl` repo, with old and new SHASUMs.
- Watch the output for the leaf process, as it gets requests, and sends completion information after 10 seconds.
- Watch the worker process as it finishes jobs and tells you that it set a commit status.
- Turn things off and on again and have fun.