# Dynv6 REST API for [`libdns`](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/ZxwyProject/libdynv6)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Dynv6 REST API](https://dynv6.github.io/api-spec/), allowing you to manage DNS records.

---

Install using `go get` (Go >= 1.20)

```sh
go get -u github.com/ZxwyProject/libdynv6
```

Token is required for authorization.

You can generate one at: https://dynv6.com/keys

```go
import "github.com/ZxwyProject/libdynv6"

p := libdynv6.Provider{
    Token: `<your http token>`,
}

zs, err := p.ListZones(context.Background())
if err != nil {
    log.Fatalln(err)
}
for i, z := range zs {
    log.Printf("[%d] %s\n", i, z.Name)
}
```

Debug mode is enabled by default. You can disable it through the following actions:

```go
import "github.com/ZxwyProject/dynv6"

dynv6.Debug = false
```
