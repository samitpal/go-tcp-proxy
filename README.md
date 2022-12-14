# tcp-proxy

A small TCP tee proxy written in Go. Idea is to bifurcate every tcp connection to the proxy and send to two remote backends.
The response from the second remote is discarded while the response from the first remote is sent back to the client.

## Install

```
$ git clone https://github.com/samitpal/go-tcp-proxy.git

$ go build -o tcp-proxy cmd/tcp-proxy/main.go

```

## Usage

```
$ ./tcp-proxy --help
Usage of tcp-proxy:
  -c	output ansi colors
  -h	output hex
  -l string
    	local address (default ":9999")
  -n	disable nagles algorithm
  -r1 string
    	first remote address. The client gets response back only from the first remote. (default "localhost:80")
  -r2 string
    	second remote address. The proxy discards the response from this remote (default "localhost:8080")
  -v	display server actions
  -vv
    	display server actions and all tcp data
```

### Simple Example

We can  use `tcp-proxy` to send traffic to two redis backends listening at localhost:6379 and localhost:7379:

```
$ tcp-proxy -r1 localhost:6379 -r2 localhost:7379
go-tcp-proxy (0.0.1) proxing from :9999 to localhost:6379 and localhost:7379
```

Then test with `redis-cli`:

```
$ redis-cli -p 9999 set test 123
```
