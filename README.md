# dyndns-client

[![Build Status](https://drone.cryptic.systems/api/badges/dyndns-client/dyndns-client/status.svg)](https://drone.cryptic.systems/dyndns-client/dyndns-client)

dyndns-client is a Daemon to listen on interface notifications produced by the linux
kernel of a client machine to update one or more DNS zones.

## Usage

To start dyndns-client just run `./dyndns-client`.

## Configuration

The program is compiled as standalone binary without third party libraries. If
no configuration file available under `/etc/dyndns-client/config.json`, than
will be the burned in configuration used. If also no configuration be burned
into the source code, that the client returned an error.

The example below describes a configuration to update RRecords triggerd by the
interface `br0` for the `example.com` zone. To update the zone is a TSIG-Key
required.

```json
{
  "interfaces": [
    "br0"
  ],
  "zones": {
    "example.com": {
      "dns-server": "10.6.231.5",
      "name": "example.com",
      "tsig-key": "my-key"
    }
  },
  "tsig-keys": {
    "my-key": {
      "algorithm": "hmac-sha512",
      "name":      "my-key",
      "secret":    "asdasdasdasdjkhjk38hcn38haoü2390dndaskdTTWA=="
    }
  }
}
```
