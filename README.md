# portal-df
A golang implementation of the [Portal](https://github.com/Paroxity/portal) protocol.
> This library is still a work in progress and does not implement all of Portal's features.

## Connecting to Portal
```go
package main

import "github.com/paroxity/portal"

func main() {
    // First create a connection to portal. The address provided should be the address of a running Portal instance.
    proxy, err := portal.New("127.0.0.1:19132")
    if err != nil {
        panic(err)
    }
    // After connecting to portal, you need to run the client to be able to read messages coming from portal.
    go func() {
        if err := proxy.Run(); err != nil {
            panic(err)
        }
    }()
    // Next you need to authenticate with a unique name and the secret configured on the Portal instance.
    if err := proxy.Authenticate("name", "secret"); err != nil {
        panic(err)
    }
    // If you are connecting as a server, you will need to send portal the address of the server.
    if err := proxy.RegisterServerInfo("127.0.0.1:19133"); err != nil {
        panic(err)
    }
}
```