# Frontend behaviour

What the donation web site (`dws/`, Next.js) does against the `delphi` API.
How to build and run it is in [`dws/README.md`](../dws/README.md); the API
itself is described in [api.md](api.md).

All amounts are passed as strings; decode them to a decimal type to keep their
precision.

## polling

- token price, ETH price, campaign totals and status (`GET /donation/data`):
  on page load and every 60 seconds
- the connected wallet's data (`GET /user/data/{address}`): every
  `NEXT_PUBLIC_UDATA_REFRESH_TIME_UNCONFIRMED` milliseconds (default 60 s)

## affiliate links

A visitor may arrive through a link carrying an affiliate code:

```
https://tm.io/donate?afc=AivCuktyds0
```

The code cannot travel in the donation transaction itself (for an erc-20
donation the transaction `data` holds the `transfer` arguments), so it is
recorded when the user connects a wallet: the frontend signs the
`wallet connection` message, which includes the code (`none` without one),
and posts it to `POST /wallet/connection`. A code that is unknown, longer than
16 characters, or the user's own is turned down with a "not found" / "cannot
use your own code" message; nothing is recorded in that case.

## connected wallet

When a wallet is connected the frontend fetches the user data for its address:

```json
{
  "donations": [
    {
      "amount": "1.2345600000",
      "asset": "eth",
      "price": "0.00100",
      "status": "confirmed",
      "tokens": "1981414",
      "ts": "2023-10-21T07:38:36Z",
      "tx_hash": "0xf98c0fe5c1bf72cad294c5ba60ca5b62d3d519fe7cc71cc3cdfb064892fa90e2",
      "usd_amount": "1981.41"
    }
  ],
  "user_data": {
    "reward": "0",
    "staked": "0",
    "status": "none",
    "tokens": "1981414",
    "total": "1981.41",
    "ts": "2023-10-21T08:28:38.744069Z"
  }
}
```

`donations` is the (paginated) donation history, oldest first; `user_data`
is the summary. An address without donations gets an empty `donations` array
and `"0"` / `none` in `user_data`. `staked`, `reward` and `status` are not used
by the backend (always `0` / `none`), see [staking](#planned-not-implemented-staking).

## donate

The user picks the asset (ETH, USDT or USDC) and the amount; the form shows the
tokens that amount buys (see [token amounts](#token-amounts)). On "donate" the
frontend

1. refuses unless the campaign status is `open` (it shows `paused` / `closed`
   otherwise);
1. refuses unless the receiving address served by `delphi` equals the one the
   build was configured with (`NEXT_PUBLIC_DONATE_PUBKEY`);
1. builds the transaction for the target chain (`NEXT_PUBLIC_TARGET_NETWORK_ID`)
   from the exact base-unit amount -- an amount with more decimals than the
   token has is rejected -- and sends it through the wallet (MetaMask or
   WalletConnect);
1. waits for the receipt and thanks the user only if it reports success.

Donations from smart-contract wallets (Safe, account-abstraction wallets) are
not credited, see [what counts as a donation](architecture.md#what-counts-as-a-donation);
the form says so.

## token amounts

Tokens are rounded up. For an ETH donation:

    ceiling(DA * EP / TP)

for a stable coin donation:

    ceiling(DA / TP)

where `DA` is the donation amount, `EP` the ETH price in USD and `TP` the token
price.

## planned, not implemented: staking

The original product spec has the connected wallet stake its tokens: a user
who has not staked any tokens may do so, and one who has may unstake them,
which takes **7 days** from the time the unstaking was requested. Nothing in
the backend or the web site implements this; the `staked`, `reward` and
`status` (`none` / `staking` / `unstaking`) fields of `user_data` are the
placeholders for it.
