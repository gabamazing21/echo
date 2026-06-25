---
slug: project-tcp-echo
step: 3
title: "Project: a TCP echo server & client"
summary: Build a TCP echo server on :9000 with an accept loop and a goroutine per connection, plus a client — the exact shape the checker grades.
est_min: 300
position: 3
---

# Project: a TCP echo server & client

> **Step 3 · How computers & networks work · Week 1 · Project**
> Concept: *sockets, listeners, connections*

You've read about how packets travel and how TCP gives you a reliable, ordered
byte stream. Now you *make one*. An **echo server** is the "hello world" of
network programming: a client connects, sends bytes, and the server sends the
exact same bytes back. It's tiny — but it forces you to meet, by hand, the three
objects every networked Go program is built from: a **listener**, a
**connection**, and the **goroutine** that services it.

## What you'll build

A TCP echo server that listens on port **9000**, and a small client to talk to
it:

- the **server** opens a listening socket, accepts connections in a loop, and
  handles each connection on its own goroutine, copying everything it reads back
  to the sender;
- the **client** dials the server, sends a line, and prints whatever comes back.

By the end you'll run the server in one terminal and this in another:

```bash
go run ./client
# type: hello
# server replies: hello
```

…and you'll have written, by hand, the same accept loop that lives inside every
HTTP server, database, and RPC framework you'll ever use.

## Why this matters

A **socket** is just an endpoint of a two-way connection, named by an IP address
and a port. Everything networked — your browser, Postgres, the Consensus app
itself — is sockets underneath. Once you've written an accept loop and a
per-connection goroutine *yourself*, `net/http` stops being magic: it's this
project with a request parser bolted on.

It matters here for a second reason: **this app's Step 3 checker is a TCP echo
server.** The grader opens a connection to your server, writes some bytes, and
asserts it gets the identical bytes back. Build it correctly here — listener,
accept loop, goroutine per connection, clean shutdown of each connection — and
you'll pass the checker on the first submission. You're building the thing the
platform is about to grade.

## 1. Set up the module

```bash
mkdir tcpecho && cd tcpecho
go mod init example.com/tcpecho
```

Give the server and client their own directories so each is its own `main`
package and builds to its own binary:

```
tcpecho/
  server/main.go
  client/main.go
```

Everything we need is in the standard library's [**`net`
package**](https://pkg.go.dev/net) — no dependencies.

## 2. Listen on a port

A **listener** is a socket bound to an address that waits for incoming
connections. You create one with `net.Listen`, passing the network (`"tcp"`) and
an address (`host:port`). Leaving the host empty (`":9000"`) means "listen on all
of this machine's interfaces, port 9000."

```go
package main

import (
	"log"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	log.Println("echo server listening on :9000")

	// accept loop goes here (step 3)
}
```

- `net.Listen` returns a `net.Listener`. If the port is already in use, you get
  an error here — that's why we check it immediately.
- `defer ln.Close()` releases the port when `main` returns.

Run it now (`go run ./server`). It should print the log line and then sit there,
doing nothing — because we haven't accepted anything yet.

## 3. The accept loop

A listener does not handle data; it only hands you **connections**. You call
`ln.Accept()` in a loop. Each call **blocks** until a client connects, then
returns a `net.Conn` — a single live connection you can read from and write to.

```go
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(conn) // one goroutine per connection — don't block the loop
	}
```

This is the heartbeat of every server on earth: **accept, hand off, accept
again.** The `go handle(conn)` is the crucial move — if you handled the
connection inline here, the loop couldn't accept a *second* client until the
first one finished. Spinning up a goroutine lets the loop immediately go back to
waiting, so thousands of clients can be served at once. Goroutines are cheap
(a few KB of stack); this pattern scales further than you'd guess.

> **Your turn:** before you write `handle`, predict what happens if you *remove*
> the `go` and call `handle(conn)` directly. Then test it: connect with two
> clients at once and watch the second one wait. Feeling that difference is the
> whole point of this step.

## 4. Handle one connection: echo with `io.Copy`

`handle` runs on its own goroutine. A `net.Conn` is both an `io.Reader` and an
`io.Writer`, so "read bytes and write the same bytes back" is *literally*
copying the connection into itself. The standard library has one function for
exactly that: [**`io.Copy(dst, src)`**](https://pkg.go.dev/io#Copy), which reads
from `src` and writes to `dst` until `src` hits EOF.

```go
func handle(conn net.Conn) {
	defer conn.Close() // always close the connection when we're done
	log.Printf("connected: %s", conn.RemoteAddr())

	// io.Copy(conn, conn) reads from the connection and writes back to it,
	// until the client closes their end (EOF). That's a complete echo server.
	if _, err := io.Copy(conn, conn); err != nil {
		log.Printf("copy: %v", err)
	}

	log.Printf("closed: %s", conn.RemoteAddr())
}
```

Add `"io"` to the server's imports. That's it — `io.Copy(conn, conn)` is a
fully working echo server in one line. When the client closes its side, `Read`
returns `io.EOF`, `io.Copy` returns cleanly, the `defer` closes our side, and the
goroutine exits.

> **Your turn:** `io.Copy` echoes raw bytes with no idea of "lines." Swap it for
> a [**`bufio`**](https://pkg.go.dev/bufio) version that reads one line at a time
> and writes it back — `bufio.NewReader(conn)` with `ReadString('\n')` in a loop,
> writing each line back with `conn.Write`. You'll need this for the line-based
> protocol stretch goal, and it teaches you what `io.Copy` is hiding.

## 5. The client

Test it without the client first: `go run ./server` in one terminal, then in
another `nc localhost 9000` (netcat). Type a line, press Enter, see it echoed.
That proves the server works.

Now write a real client. Dialing is the mirror of listening: `net.Dial` opens a
connection *to* an address and returns the same `net.Conn` type.

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Read a line the user types on stdin...
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		log.Fatalf("read stdin: %v", err)
	}

	// ...send it to the server...
	if _, err := fmt.Fprint(conn, line); err != nil {
		log.Fatalf("write: %v", err)
	}

	// ...and print whatever the server echoes back.
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Fatalf("read reply: %v", err)
	}
	fmt.Print("server replied: ", reply)
}
```

Run the server, then `go run ./client`, type a line, and watch it bounce back.
You have just written both halves of a network conversation.

> **Your turn:** wrap the client's send/receive in a `for` loop so you can type
> many lines in one session, exiting on an empty line. The server already
> supports it — connections are long-lived until *you* close them.

## Stretch goals

Pick at least one. This is where a toy becomes a real server.

- **Graceful shutdown.** Catch `SIGINT`/`SIGTERM` with `os/signal` and
  `signal.NotifyContext`. On signal, stop accepting (close the listener) and let
  in-flight connections drain instead of slamming them shut. Real servers must
  not lose data mid-request.
- **Track concurrent clients.** Keep a counter of live connections (protect it
  with a `sync.Mutex`, or use `sync/atomic`) and log it on every connect and
  disconnect. Connect three netcats at once and watch the count rise and fall —
  proof your goroutine-per-connection model really is concurrent.
- **A line-based protocol.** Move past raw echo: read line by line and act on
  commands. `PING` → `PONG`, `TIME` → the current time, `QUIT` → close the
  connection, anything else → echo it. This is the seed of every text protocol
  (Redis, SMTP, IRC) you'll ever read about.
- **Timeouts.** Use `conn.SetReadDeadline` so an idle client is dropped after,
  say, 30 seconds. Without deadlines a stalled client holds a goroutine forever —
  a classic resource leak.
- **Concurrency-safe broadcast (hard).** Turn it into a tiny chat server: every
  line a client sends is echoed to *all* connected clients. Now you need a shared
  registry of connections and careful locking — your first taste of the shared-state
  problems Step 4 is all about.

## Free resources

- [**Go `net` package docs**](https://pkg.go.dev/net) — read `Listen`, `Dial`,
  `Listener`, and `Conn`. This is the whole project's API surface; it's short.
- [**Beej's Guide to Network Programming**](https://beej.us/guide/bgnet/) — the
  classic, free, deeply readable intro to sockets. It's in C, but the *concepts*
  (sockets, bind/listen/accept, the client/server dance) are exactly what you
  just built — read the "What is a socket?" and "Client-Server Background"
  sections.
- [**`io` package docs**](https://pkg.go.dev/io) — understand `Reader`, `Writer`,
  and why `io.Copy(conn, conn)` is all it takes.

## Done when

- [ ] `go run ./server` listens on `:9000` and logs each connect/disconnect.
- [ ] The server uses an **accept loop** with **one goroutine per connection** —
      a second client connects without waiting for the first.
- [ ] Bytes sent are echoed back **byte-for-byte** (verified with `nc` *and* your
      own client) — and it **passes the Step 3 checker in the app.**
- [ ] Every connection is closed with `defer conn.Close()`; no goroutine leaks.
- [ ] Your client dials, sends a line, and prints the echoed reply.
- [ ] You attempted at least one stretch goal.

When the checker turns green, commit this project on your **Roadmap**. You've now
built a concurrent network server from raw sockets — the foundation under every
distributed system in the steps ahead. Next up: **Step 4, concurrency and shared
state.**

## Common interview gotchas

- **Graceful shutdown = "stop accepting" *then* "drain", with a deadline.** On SIGTERM, **close the listener** (so `Accept` returns and the loop exits — no new connections), then `WaitGroup.Wait()` for in-flight connections, bounded by a timeout so a stuck connection can't hang shutdown forever. Distinguishing *stop accepting* from *finish existing* is the signal.
- **TCP is a byte stream, not messages.** One `conn.Read` can return half a message, exactly one, or several concatenated — TCP has **no message boundaries**. Any real protocol needs **framing**: a delimiter (`bufio.Scanner`) or a length prefix (`io.ReadFull`). Raw `io.Copy(conn,conn)` works for echo *only* because it ignores boundaries.
- **No deadlines = goroutine + FD leak.** A client that connects and sends nothing blocks `Read` forever, leaking a goroutine per idle peer (slowloris-style DoS). Use `SetReadDeadline`/`SetWriteDeadline`, re-armed each iteration — note they're **absolute times**, not durations.
- **Defend goroutine-per-connection first, *then* name the C10k/C1M limits.** The Go netpoller (epoll/kqueue) parks blocked reads off-thread, so this model scales to ~100k+. The real ceilings are **memory** (~2–8 KB/goroutine × 1M = GBs), **file descriptors** (`ulimit -n`), and **GC/scheduler pressure** — mitigate with buffer pooling and connection caps before reaching for an explicit event loop.
