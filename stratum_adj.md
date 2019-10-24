# Adjusted Stratum for Pegnet

The pool uses an adjusted form of the Stratum protocol. This uses raw tcp with line-based communication and json-rpc encoding. This provides easy extensibility and debugging in the early days of the pool. If bandwidth ever becomes a concern, alternative encoding schemes can be supported.

Stratum can be found:
- https://slushpool.com/help/stratum-protocol#compatibility
- https://en.bitcoin.it/wiki/Stratum_mining_protocol
- https://slushpool.com/help/topic/stratum-protocol/
- https://github.com/aeternity/protocol/blob/master/STRATUM.md#mining-configure
- https://github.com/ctubio/php-proxy-stratum/wiki/Stratum-Mining-Protocol
- https://github.com/str4d/zips/blob/77-zip-stratum/drafts/str4d-stratum/draft1.rst#rationale
    - https://github.com/str4d/zips/blob/77-zip-stratum/drafts/str4d-stratum/draft1.rst#protocol-flow


# Methods (client to server)

## mining.authorize

request
```json
{
  "method" : "mining.authorize",
  "id": 0,
  "params": ["username,minerid", "password", "invite-code", "payout-address"]
}
```
Note: password, invite-code, and payout addresses are optional once the username has already been successfully authorized by the pool.

response
```json
{
  "id": 0,
  "result": true,
  "error": null
}
```
The result from an authorize request is usually true (successful), or false. The password may be omitted if the server does not require passwords. Invite code, password, and payout address should typically only be provided upon the very first authentication for a given username, as they are ignored on subsequent authorize calls.


## mining.get_oprhash

request
```json
{
  "method" : "mining.get_oprhash",
  "id": 0,
  "params": ["jobID"]
}
```
Server should send back an array with the Oracle Price Record hash for the given job id.


## mining.submit

request
```json
{
  "method" : "mining.submit",
  "id": 0,
  "params": ["username", "jobID", "nonce", "oprHash", "target"]
}
```
Miners submit shares using the method "mining.submit". Client submissions contain:

1) Worker Name
2) Job ID
3) Nonce
4) OPR hash
5) Target

Server response is result (true for accepted, false for rejected). Alternatively, you may receive an error with more details.


## mining.subscribe

request
```json
{
  "method" : "mining.subscribe",
  "id": 0,
  "params": ["user-agent/version"]
}
```

response
```json
{
  "id": 0,
  "error": null,
  "result": ["sessionID", "nonce"],
}
```

The client receives a result:
```json
[[["mining.set_target", "subscription id 1"], ["mining.notify", "subscription id 2"]], "nonce"]
```
The result contains two items:


1) Subscriptions - An array of 2-item tuples, each with a subscription type and id.
2) Nonce - Hex-encoded, per-connection unique string which will be used for creating generation transactions later.

## mining.suggest_target

request
```json
{
  "method" : "mining.suggest_target",
  "id": 0,
  "params": ["preferred-target"]
}
```
Used to indicate a preference for mining target to the pool. Servers are not required to honor this request.


# Methods (server to client)

## client.get_version

request
```json
{
  "method" : "client.get_version",
  "id": 0,
  "params": null
}
```

response
```json
{
  "id": 0,
  "error": null,
  "result": ["name", "version"],
}
```
The client should send a result String with its name and version.


## client.reconnect

request
```json
{
  "method" : "client.reconnect",
  "id": 0,
  "params": ["hostname", "port", "waittime"]
}
```
The client should disconnect, wait waittime seconds (if provided), then connect to the given host/port (which defaults to the current server). Note that for security purposes, clients may ignore such requests if the destination is not the same or similar.


## client.show_message

request
```json
{
  "method" : "client.show_message",
  "id": 0,
  "params": ["message"]
}
```
The client should display the message to its user in a human-readable way.


## mining.notify

request
```json
{
  "method" : "mining.notify",
  "id": 0,
  "params": ["jobID", "oprHash", "CLEANJOBS"]
}
```
Fields in order:

1) Job ID. This is included when miners submit a results so work can be matched with proper transactions.
2) Oracle Price Record hash. Used to build the header.
3) (optional) "CLEANJOBS". Used to force the miner to begin using a new mining parameters immediately.


## mining.set_target

request
```json
{
  "method" : "mining.set_target",
  "id": 0,
  "params": ["target"]
}
```
The server can adjust the target required for miner shares with the "mining.set_target" method. The miner should begin enforcing the new target on the next job received. Some pools may force a new job out when set_target is sent, using CLEANJOBS to force the miner to begin using the new target immediately.


## mining.set_nonce

request
```json
{
  "method" : "mining.set_nonce",
  "id": 0,
  "params": ["nonce"]
}
```
This value, when provided, replaces the initial subscription value beginning with the next mining.notify job.


## mining.stop_mining

request
```json
{
  "method" : "mining.stop_mining",
  "id": 0,
  "params": null
}
```
Instructs miner to pause mining until a new job is received.


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