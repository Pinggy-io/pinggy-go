# Pinggy

Pinggy provides tunneling service over ssh. It provides different types of tunnel. This module provides a easy API for Pinggy.

This module is essentially a wrapper over golang.org/x/crypto/ssh. However, it makes tunnel creation as simple as
```
listener, _ := pinggy.Connect()
```

The sdk exposes multiple helpfull APIs. It allows users to use the special feature `sshOverSsl`.


## Documentation

https://pkg.go.dev/github.com/Pinggy-io/pinggy-go/pinggy