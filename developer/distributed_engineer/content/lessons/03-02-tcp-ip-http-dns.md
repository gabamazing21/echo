---
slug: net-tcp-ip-http-dns
step: 3
title: How two machines talk — TCP/IP, HTTP, DNS
summary: Trace a request from URL to bytes on the wire — the layered stack, IP routing, TCP vs UDP, ports, DNS, and HTTP.
est_min: 420
position: 2
---

# How two machines talk — TCP/IP, HTTP, DNS

> **Step 3 · How computers & networks work · Week 1**
> Concept: *the network stack, request/response*

You can write Go, and you understand how a single machine works. But a
distributed system isn't one machine — it's *many* machines talking to each other
over a network that can be slow, lossy, and occasionally lying. Before you can
reason about consensus, replication, or fault tolerance, you need a clear mental
model of what actually happens when one machine sends bytes to another. This
lesson builds that model from the wire up.

## Why this matters

Here's the uncomfortable truth at the heart of this whole course: **a distributed
system is just machines talking over an unreliable network.** Every hard problem
you'll meet later — split brain, stale reads, partitions, retries that
double-charge a customer — traces back to the network not behaving like a local
function call. In Step 7 you'll meet the *Eight Fallacies of Distributed
Computing*, the first of which is "the network is reliable." It isn't. The only
way to build systems that survive that is to understand, concretely, what's
happening underneath. So we start here, with two machines and a wire.

## 1. The stack is layered — and that's the whole trick

Nobody could build the internet as one giant program. Instead it's split into
**layers**, each solving one problem and handing off to the next. Each layer
talks to its peer on the other machine as if the layers below didn't exist.

| Layer | Job | Examples | Unit |
|---|---|---|---|
| **Application** | What the bytes *mean* | HTTP, DNS, SMTP | message |
| **Transport** | Get data to the right *program*, reliably or not | TCP, UDP | segment |
| **Internet (IP)** | Get a packet to the right *machine*, anywhere | IPv4, IPv6 | packet |
| **Link** | Move bits across one physical hop | Ethernet, Wi-Fi | frame |

The key idea is **encapsulation**: your HTTP request gets wrapped in a TCP
segment, which gets wrapped in an IP packet, which gets wrapped in a link frame.
Each layer adds its own header — like nested envelopes. At the destination, each
layer peels off its envelope and hands the contents up. You write to the top; the
stack handles everything below.

## 2. IP: addressing and routing

The **Internet Protocol** gives every machine an **IP address** and defines how a
packet finds its way there. An IPv4 address is four bytes — `93.184.216.34`. IPv6
addresses are 16 bytes, written in hex — `2606:2800:220:1:248:1893:25c8:1946` —
because we ran out of the ~4 billion IPv4 addresses.

IP is **best-effort and connectionless**. A packet is dropped into the network
with a destination address, and **routers** pass it hop by hop toward that
address — each router only needs to know "which direction is closer," not the
whole path. Crucially, IP makes **no promises**: packets can be lost, duplicated,
delayed, or arrive out of order. There's no "connection" at this layer at all —
just independent packets. Making that chaos *reliable* is the next layer's job.

## 3. TCP vs UDP: two ways to use ports

Both transport protocols introduce **ports** — a 16-bit number (0–65535) that
says *which program* on the machine should get the data. An IP address finds the
machine; the port finds the process. A web server listens on port 80 (HTTP) or
443 (HTTPS); DNS uses 53. The pair `IP:port` is a **socket**, and a connection is
identified by the four-tuple `(src IP, src port, dst IP, dst port)`.

**TCP** (Transmission Control Protocol) is **connection-oriented, reliable, and
ordered.** It turns IP's unreliable packets into a clean, ordered byte stream:

- **Reliable** — every byte is acknowledged; lost data is retransmitted.
- **Ordered** — bytes arrive in the order you sent them, reassembled from
  out-of-order packets.
- **Flow & congestion control** — it slows down so it doesn't overwhelm the
  receiver or the network.

**UDP** (User Datagram Protocol) is **fire-and-forget.** You send a datagram and
hope it arrives. No connection, no acknowledgements, no ordering, no
retransmission. Why would you want that? Because reliability costs latency. For
DNS lookups, live video, and gaming, sending a fresh packet *now* beats waiting
to redeliver a stale one.

| | TCP | UDP |
|---|---|---|
| Connection | yes (handshake first) | none |
| Reliable delivery | yes | no |
| Ordering | yes | no |
| Overhead / latency | higher | lower |
| Use it for | HTTP, databases, SSH | DNS, video, games |

Most of what you build in this course rides on TCP, because correctness usually
beats raw speed — but knowing *why* the choice exists is the point.

## 4. The TCP 3-way handshake

A TCP connection doesn't just *start* — both sides must agree to it first, by
exchanging three packets. Each side picks a random starting **sequence number**
and confirms it sees the other's:

```
Client                        Server
  |  ---- SYN (seq=x) -------->  |   "I'd like to talk; my seq starts at x"
  |  <-- SYN-ACK (seq=y,ack=x+1) |   "OK; my seq starts at y, I got your x"
  |  ---- ACK (ack=y+1) ------>  |   "Got your y. We're connected."
  |                              |
  |  <====== data flows =======> |
```

After these three messages (**SYN**, **SYN-ACK**, **ACK**), both sides have
confirmed they can send *and* receive, and they've agreed on sequence numbers so
every byte can be tracked and reordered. This is also why opening a connection
has a cost: at least one network round-trip *before* any real data moves. Over a
long-distance link that's tens of milliseconds you pay up front — one reason
systems pool and reuse connections instead of opening a new one per request.

## 5. DNS: turning names into addresses

You don't type `93.184.216.34` — you type `example.com`. The **Domain Name
System** is the internet's phone book, translating human-friendly names into IP
addresses. It's a distributed, hierarchical lookup (a real distributed system in
its own right). Roughly:

1. Ask the **root** servers: "who handles `.com`?"
2. Ask the `.com` servers: "who handles `example.com`?"
3. Ask `example.com`'s **authoritative** server: "what's the A record?"

In practice your machine asks a **resolver** (often your ISP's or `8.8.8.8`),
which does this walk for you and **caches** the answer with a TTL so the next
lookup is instant. DNS usually runs over **UDP** — it's a small request and a
small reply, and if a packet drops you just ask again.

In Go you rarely speak DNS by hand; the standard library resolves names for you.
But you can do a lookup explicitly:

```go
package main

import (
	"fmt"
	"net"
)

func main() {
	addrs, err := net.LookupHost("example.com")
	if err != nil {
		fmt.Println("lookup failed:", err)
		return
	}
	for _, a := range addrs {
		fmt.Println(a) // e.g. 93.184.216.34
	}
}
```

`net.LookupHost` returns the IP addresses a name resolves to — the very first
step the next section automates for you.

## 6. HTTP: the application riding on top of TCP

**HTTP** (HyperText Transfer Protocol) is a simple **request/response** protocol
that runs *over* a TCP connection. The client opens a TCP connection to the
server's port (80 or 443), sends a text request, and reads a response. A raw
HTTP/1.1 request is just lines of text:

```
GET /index.html HTTP/1.1
Host: example.com
User-Agent: curl/8.0

```

And the response looks like:

```
HTTP/1.1 200 OK
Content-Type: text/html
Content-Length: 1256

<!doctype html> ...
```

A **request** has a method (`GET`, `POST`, …), a path, headers, and an optional
body. A **response** has a status code (`200 OK`, `404 Not Found`, `500 Internal
Server Error`), headers, and a body. That's the whole shape — and it's the shape
every web API you'll ever build or call is made of.

Notice the layering pay-off: HTTP doesn't care about lost packets or reordering —
**TCP** already guaranteed an ordered, reliable byte stream. HTTP just reads and
writes text. Each layer trusts the one below.

Here's a sketch of opening a raw TCP connection in Go — what `net/http` does for
you under the hood:

```go
package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	// Dial = DNS lookup + TCP 3-way handshake, all in one call.
	conn, err := net.Dial("tcp", "example.com:80")
	if err != nil {
		fmt.Println("dial failed:", err)
		return
	}
	defer conn.Close()

	// Write a raw HTTP/1.1 request. \r\n ends each line; a blank line ends headers.
	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")

	// Read the first line of the response — the status line.
	status, _ := bufio.NewReader(conn).ReadString('\n')
	fmt.Print(status) // HTTP/1.1 200 OK
}
```

In real code you'd use `http.Get`, which handles DNS, the handshake, headers,
redirects, and TLS for you. But seeing the raw bytes once makes it concrete: HTTP
is just text over a TCP socket.

## 7. The full journey: typing a URL to bytes on the wire

Let's put every layer together. You type `http://example.com/index.html` and hit
enter. Here is everything that happens:

1. **Parse the URL** → scheme `http`, host `example.com`, path `/index.html`,
   default port `80`.
2. **DNS resolution** — the resolver turns `example.com` into an IP like
   `93.184.216.34` (a UDP request to port 53, often answered from cache).
3. **TCP handshake** — your machine opens a connection to `93.184.216.34:80` via
   the SYN / SYN-ACK / ACK exchange (section 4).
4. **Send the HTTP request** — `GET /index.html HTTP/1.1` plus headers, written
   into the TCP stream. The stack wraps it: HTTP → TCP segment → IP packet → link
   frame, and routers carry the packets across the internet hop by hop.
5. **Server responds** — it sends back `200 OK` and the HTML body over the same
   connection; TCP reassembles the packets into an ordered stream for you.
6. **Render / close** — the browser parses the HTML (and fetches more resources,
   each its own request); the TCP connection is closed or kept alive for reuse.

Every single step can fail or stall — DNS times out, the handshake never
completes, a packet is dropped mid-stream, the server is partitioned away. That
fragility *is* the subject of distributed systems. You're not learning networking
as trivia; you're learning the medium your future systems live and die in.

## Do it yourself

1. Run the `net.LookupHost` program against three domains (`example.com`,
   `go.dev`, your own site). Notice some names return multiple IPs — that's load
   balancing and redundancy.
2. Run the raw `net.Dial` HTTP sketch. Then try it against a host that doesn't
   exist and watch *where* it fails (DNS vs dial vs read).
3. From a terminal, run `curl -v http://example.com` and read the verbose output:
   you'll literally see the DNS resolution, the connection, and the request/
   response headers — every layer from this lesson, live.
4. Run `dig example.com` (or `nslookup example.com`) and identify the A record and
   the TTL.
5. Read [**Beej's Guide to Network Programming**](https://beej.us/guide/bgnet/) —
   the friendliest deep intro to sockets, TCP/UDP, and what `Dial`/`Listen` do
   under the hood. Read at least the "What is a socket?" and client/server
   sections.
6. Read the opening chapters of [**High Performance Browser
   Networking**](https://hpbn.co/) (free online) — *Primer on Latency and
   Bandwidth*, *Building Blocks of TCP*, and *Building Blocks of UDP*. This is the
   best free explanation of *why* the handshake and round-trips cost what they do.
7. For the rigorous version, skim the [**Stanford CS144**](https://cs144.github.io/)
   course site — its lectures build the whole stack from the link layer up. You
   don't need to do the labs now; just absorb the layering model.

## Check yourself

You're ready to move on when you can answer, *without looking*:

- Name the four layers of the stack and the one job each one does.
- What does an IP address identify, and what does a port identify?
- Give three concrete differences between TCP and UDP, and one good use for each.
- Walk through the TCP 3-way handshake and explain *why* it costs a round-trip
  before any data moves.
- What does DNS do, and why does it usually run over UDP?
- Trace, in order, everything that happens between typing a URL and the first
  bytes of HTML coming back.

When all six feel obvious, commit this task on your **Roadmap** and take the
Step 3 quiz. Next up: **how the internet routes at scale**, then onward toward the
fallacies that make distributed systems hard.

## Common interview gotchas

- **"What happens when you type a URL" rewards *depth*, not the list.** Hit every layer (parse → DNS → TCP handshake → **TLS handshake if HTTPS** → request → response → render), and call out **caching at each level** and **connection reuse**. Forgetting TLS, or treating DNS/TCP/TLS as one step, is the common miss.
- **Why *three* messages in the TCP handshake.** Each side picks a random initial sequence number and needs the other to acknowledge it — two messages would leave one direction's ISN unconfirmed. Random ISNs also fend off stale/spoofed segments. The cost is **one full round trip before any data**, which is *why* you pool connections.
- **TCP vs UDP is a reasoning question, not a table.** The signal is "is a late retransmit worth more than the freshest packet?" → TCP for correctness (HTTP, DBs, SSH), UDP for fresh-now (video, games, DNS). Bonus: QUIC/HTTP3 rebuilds reliability *on top of* UDP to dodge TCP head-of-line blocking.
- **DNS uses UDP for speed, falls back to TCP for size.** One small request/reply isn't worth a handshake; if it drops, retry. It switches to **TCP** when the response is truncated (TC bit set, historically >512 B), for zone transfers (AXFR), and for DoT/DoH.
- **TLS cost is *round-trip* bound.** TCP handshake **plus** TLS (~2 RTT in 1.2, **1 RTT in 1.3**, 0-RTT on resumption) before any HTTPS byte. The wins come from doing fewer handshakes — keep-alive, session resumption, HTTP/2 multiplexing, OCSP stapling.
