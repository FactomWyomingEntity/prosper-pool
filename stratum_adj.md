# Adjusted Stratum for Pegnet

The pool will use an adjusted form of the stratum protocol. The pool will use raw tcp with line based communication and json-rpc encoding. This will provide easy extensibility and debugging in the early days of the pool. If bandwidth ever becomes a concern, alternative encoding schemes can be supported.

Stratum can be found:
- https://slushpool.com/help/stratum-protocol#compatibility
- https://en.bitcoin.it/wiki/Stratum_mining_protocol
- https://slushpool.com/help/topic/stratum-protocol/
- https://github.com/aeternity/protocol/blob/master/STRATUM.md#mining-configure
- https://github.com/ctubio/php-proxy-stratum/wiki/Stratum-Mining-Protocol


# Methods

```json
{
  "method" : "",
  "id": 0,
  "params": null
}
```

## mining.subscribe

request
```json
{
  "method" : "mining.subscribe",
  "id": 0,
  "params": ["user agent/version", "password"]
}
```

## mining.authorize

request
```json
{
  "method" : "mining.authorize",
  "id": 0,
  "params": ["username", "password"]
}
```

response
```json
{
  "id": 0,
  "result": true,
  "error":null
}
```

# Error codes

```json
    {
      "..." : "...",
      "error": {
        "code": 0,
        "message": "short desc",
        "data" : "anything"
      }
    }
```

```bash
-1, Unknown exception, error message should contain more specific description
-2, “Service not found”
-3, “Method not found”
-10, “Fee required”
-20, “Signature required”, when server expects request to be signed
-21, “Signature unavailable”, when server rejects to sign response
-22, “Unknown signature type”, when server doesn’t understand any signature type from “sign_type”
-23, “Bad signature”, signature doesn’t match source data

```