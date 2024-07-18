# Fonzie 👍 
<img width=150 src="https://c.tenor.com/VOblnhsOkY4AAAAd/thumbs-up-smug.gif">

The interchain cosmos faucet for discord.

* Disambiguates by bech32 prefix
* Supports multiple chains at once & rate-limiting
* State-of-the-art emoji response technology. Inspired by the fonz [👍](https://en.wikipedia.org/wiki/Fonzie)

## Building
```bash
go build .
```

## Usage

### Environment Variables

* `BOT_TOKEN` -- [Create a Discord token](https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token)
* `MNEMONIC`  -- 12 or 24 word seed string, shared for each chain
* `CHAINS`    -- A JSON object, keyed by each bech32 prefix, value is a RPC endpoint
* `FUNDING`   -- Similar to CHAINS, value is how much funding to sip with each tap

#### An example configuration supporting Umee, Atom, Juno & Osmosis

```bash
BOT_TOKEN='<discord bot token>'
MNEMONIC='<12 or 24 word mnemonic>'
CHAINS='[{"prefix":"umee","rpc":"https://rpc.alley.umeemania-1.network.umee.cc:443"},{"prefix":"cosmos","rpc":"https://rpc.flash.gaia-umeemania-1.network.umee.cc:443"},{"prefix":"juno","rpc":"https://rpc.section.juno-umeemania-1.network.umee.cc:443"},{"prefix":"osmo","rpc":"https://rpc.wall.osmosis-umeemania-1.network.umee.cc:443"}]'
FUNDING='{"umee":"100000000uumee","cosmos":"100000000uatom","juno":"100000000ujuno","osmo":"100000000uosmo"}'
```

### Running

```bash
./fonzie
```

### Bot Commands

See [help.md](help.md).  This file is rendered for the `!help` command.

## Screenshots

<img width="596" alt="Screen Shot 2022-04-08 at 12 49 55 AM" src="https://user-images.githubusercontent.com/42952/162380395-81da39af-f88c-4579-a02a-3188a886be90.png">

## Releasing

### Example
1. Suppose you want to release version `v99.99.99`
2. Make sure you are on main and have a clean git state
3. `bin/release v99.99.99`

#### Output
```
Everything up-to-date
Total 0 (delta 0), reused 0 (delta 0), pack-reused 0
To github.com:umee-network/fonzie.git
 * [new tag]         v99.99.99 -> v99.99.99
```
