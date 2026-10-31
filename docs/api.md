# API contract

`delphi` serves the REST API the frontend uses. The spec is
[`api/delphi.yaml`](../api/delphi.yaml); `make codegen` generates
`api/server.go` and `api/models.go` from it, and every request is validated
against it before it reaches a handler.

| endpoint | auth | purpose |
|---|---|---|
| `GET /donation/data` | -- | token and ETH price, campaign totals and status, receiving address |
| `GET /user/data/{address}` | -- | an address' totals and paginated donation history |
| `POST /wallet/connection` | signed | record a wallet connection, with the affiliate code it came with |
| `POST /affiliate/code` | signed | return (or create) the caller's affiliate code |
| `GET /live`, `/ready`, `/version` | -- | health and version; exempt from rate limiting |

## authentication

A signed request carries three headers:

- `delphi-key` -- the caller's address (`0x` + 40 hex digits)
- `delphi-ts` -- a unix timestamp (1 to 12 decimal digits)
- `delphi-signature` -- a wallet (`personal_sign`) signature (`0x` + 130 hex
  digits)

Anything else is a 400 before the handler runs; the request line and headers
together are capped at 8 KB. The timestamp may be at most
`DWS_MAX_TIMESTAMP_AGE` old (default 30 s, see
[configuration](configuration.md)) and at most 5 s in the future.

The signature is taken over a message built from the words of the endpoint
path, the lower cased address and the timestamp:

```
affiliate code, 0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154, 2023-10-23 18:45:19+00:00
```

`/wallet/connection` appends the affiliate code from the request body, exactly
as sent (`none` without one):

```
wallet connection, 0xded1fe6b3f61c8f1d874bb86f086d10ffc3f0154, 2023-10-23 18:45:19+00:00, none
```

So a signature is bound to its endpoint, its account and -- on
`/wallet/connection` -- its affiliate code: a harvested signature cannot be
replayed inside the timestamp window for another account or with another
code. Only the canonical (EIP-2 low `s`) form of a signature is accepted, with
a recovery id of 27/28 or 0/1; `(r, N-s)` recovers the same address but is
answered with a 401. Wallets sign with low `s`.

The message format is shared with the frontend: **deploy `delphi` and the
frontend together** whenever it changes, or every signed request is rejected
with a 401.

## `GET /user/data/{address}`

The endpoint is unauthenticated and donation addresses are public on chain, so:

- **the donation history is paginated.** `limit` (1..100, default 50) and
  `offset` (0..10000, default 0) select the page, oldest donation first. The
  caps keep a single request from turning a few bytes of input into megabytes
  of database read and egress (postgres reads every skipped row); an `offset`
  above 10000 is rejected with a 400.
- **the affiliate code is not part of the response.** It would let anyone walk
  the donor set into a referral graph; it is only served over `POST
  /affiliate/code`, signed by the address it belongs to.

`donations` is always an array (empty for no donations or a page past the end
of the history). An address without data is reported with `"0"` amounts and
`status: "none"`.

## `POST /wallet/connection`

The address in the body must be the signing address. A connection whose
affiliate code is turned down records nothing and says why:

- an unknown code -- `422`, code `119`
- a *self-referral*, i.e. the caller's own code -- `422`, code `120`
  (attaching another wallet's code is the normal referral)

The frontend tells the user why the referral did not take.

## errors

An error response carries a numeric code and a fixed, generic message. The
underlying error -- which names tables, columns and the driver -- is logged
server side only, so quote the code when reporting a problem.

| code | status | meaning |
|---|---|---|
| 101 | 400 | `/wallet/connection` body cannot be bound |
| 102, 103 | 400 | invalid ethereum address (body / path) |
| 104 | 500 | wallet connection could not be recorded |
| 106 | 500 | user data could not be fetched |
| 107, 113 | 400 | `delphi-ts` cannot be parsed (`/affiliate/code`, `/wallet/connection`) |
| 108, 114 | 400 | `delphi-ts` outside the accepted window |
| 109, 110 | 500 | affiliate code could not be fetched / generated |
| 111 | 500 | donation data could not be fetched |
| 112 | 500 | `DWS_DONATION_ADDRESS` not set |
| 115 | 400 | invalid affiliate code (bound body value) |
| 116 | 4xx | request validation failed (also unknown path: 404, unknown method: 405) |
| 117 | 400 | request parameters cannot be bound |
| 118 | 400 | `offset` out of range |
| 119 | 422 | unknown affiliate code |
| 120 | 422 | affiliate code belongs to the caller |

A signature that does not verify, or a body address that is not the signing
address, is a bare 401.
