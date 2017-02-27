# Queried

Forward dns with authority while still recursivly resolving.

## Why?

Tired of trying to get what we needed from off the shelf solutions, I wrote this.

## Config

See (the sample configuration file)[config.toml.example]

### resolvers `[string]`

An array of ip addresses for upstream resolvers

```
["8.8.8.8","8.8.4.4"]
```

### listen `[string]`

An array of ip's and ports to listen on

```
[":53", "127.0.0.1:5053"]
```

### local_networks `[string]`

An array of networks in CIDR notation to consider private, internal or otherwise safe to relay to

```
[
	"127.0.0.0/8",
	"192.168.0.0/16",
	"172.16.0.0/12",
	"10.0.0.0/8",
	"fc00::/7,"
	"::1/64",
]
```

### `[[forwarded_zone]]`

A zone and how to forward it

#### name `string`

Name of the zone to forward

```
"consul."
```

#### authoritative `boolean`

Return records with the `aa` flag set

```
true
```

#### upstream `string`

Server to forward to

```
"172.31.9.24:8600"
```

#### private `bool`

Answer requests for this zone only from private networks

```
true
```


## License

Copyright (c) 2017 Shannon Wynter. Licensed under GPL3. See the [LICENSE.md](LICENSE.md) file for a copy of the license.