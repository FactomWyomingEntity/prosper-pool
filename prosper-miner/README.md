# Prosper Miner


The prosper-miner binary is a PegNet miner compatible with the Prosper pool (`private-pool`) server. When a miner submits their work to the pool in the form of a share, the pool software determines whether or not to submit that share, and will credit the miner for the work they have done.

*Note: for best results, use Go v1.13*

# Usage

## Example setup
First, make sure that [a prosper pool server is running](../README.md#Usage). Then, from the `prosper-miner` directory (`cd prosper-miner`), use `go build` or `go install` to build the miner binary. Assuming that the pool server is running and accessible at the address `123.45.67.89:1234` you can connect a miner with the username `user@example.com` to it like so:

```
./prosper-miner --poolhost 123.45.67.89:1234 --user user@example.com
```

## Notes

* The username provided must be a valid email address.
* The first time you authenticate a particular username, you must provide an invite code (with `--invitecode` or `-i`) and use the `--password` (or `-p`) flag to enable a password prompt upon startup. You will also need to provide a payout address (`-a`) *Note: the password should not be provided on the command-line; the `-p` flag is simply a boolean to enable the prompt.*

An example of someone connecting to the pool for the first time using the invite-code "invite-EAXPRO" and mining with 4 threads might be like below. Once you are registered, you will only need to provide the user field.

```
./prosper-miner -s 123.45.67.89:1234 -u user@example.com -t 4 -i invite-EAXPRO -p -a FA2jK2HcLnRdS94dEcU27rF3meoJfpUcZPSinpb7AwQvPRY6RL1Q
```
The password would then be entered and confirmed at the ensuing prompt.

* Make sure to enter the right username, as this is how you will be credited for the hashrate you provide to the pool.

* The configuration for pool miners is by default stored and managed at `~/.prosper/prosper-miner.toml` though this can be changed with the `--config` command-line option.


## Command-line options

You can use `prosper-miner --help` to list the command-line arguments and options:

```
Usage:
  prosper-miner [flags]

Flags:
  -c, --config string       config path location
  -h, --help                help for prosper-miner
  -i, --invitecode string   Invite code for initial user registration
  -m, --minerid string      Minerid should be unique per mining machine (default is randomly generated)
  -t, --miners int          Number of mining threads (default 8)
  -p, --password            Enable password prompt for user registration
  -s, --poolhost string     URL to connect to the pool (default "localhost:1234")
  -u, --user string         Username to log into the mining pool

```