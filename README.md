# truth memes donation web site

## general

We want to build a 1-page web site similar to https://memeinator.com/ -- the donation widget needs to be on top of the page. It shall allow a user to donate ETH, USDT or USDC.

The backend used by the donation web site frontend is called `delphi` and it will expose a [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) to the frontend.

## how to run the dws backend

The `dws` backend consists of a `postgres` database and 5 services
- `buck`: ETH/latest crawler, checks the latest blocks for donation transactions and inserts these into the database (in state `unconfirmed`)
- `buck`: ETH/finalized crawler, checks the finalized blocks for donation transactions and confrm them, also updates the donation campaign statistics and the token price (if/as needed)
- `buck`: ETH/old-unconfirmed crawler, checks for donations that are older than 30 minutes but still unconfirmed, attempts to fetch the respective finalized blocks and confirm these donation transactions
- `pulitzer`: pulls the ETH price from 6 exchanges and inserts an average price into the database every minute
- `delphi`: [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) server -- only serves data from the database

The backend services are written in `go` -- you will thus need `go` on your development system. For testing purposes the `postgres` database can be run in a docker container i.e. you will need docker as well.

Once you have these in place you can simply run
1. `go mod tidy` to fetch the dependencies
1. `make` to build the services
1. `make run_db` to run a fresh database in a docker container
1. `make run_pulitzer` in a separate terminal
1. `make run_delphi` in a separate terminal
1. `make run_buck_latest` in a separate terminal
1. `make run_buck_final` in a separate terminal
1. `make run_buck_old_unconfirmed` in a separate terminal

All services support the `-p` command-line flag allowing you to set the port they are listening to. The default ports are as follows


|service          | port   | content served |
|:-----------------:|------:|--------------------------------------------|
|delphi      | 8080  | REST API + /live and /ready healtcheck endpoints |
|pulitzer      | 8081  | /live and /ready healtcheck endpoints |
|buck/latest      | 8082  | /live and /ready healtcheck endpoints |
|buck/final      | 8083  | /live and /ready healtcheck endpoints |
|buck/old-unconfirmed      | 8084  | /live and /ready healtcheck endpoints |

## requirements & rules
1. all amounts are passed as strings and should be decoded to a `decimal` type to preserve precision
1. users may be sent to our web site via a link that contains an affiliate code e.g.

    ```https://tm.io/donate?afc=AivCuktyds0```

    If such an affiliate code is present then it needs to be passed to the `delphi` backend after a donation transaction. We cannot capture it in the ethereum transaction if the latter involves calling an `erc-20` smart contract -- the `data` property (of the ethereum transaction) holds the arguments of the function invoked on the smart contract.

1. the frontend needs to get the token price from the `dsw` backend upon each refresh

1. the frontend needs to get the ETH price from the `dsw` backend
    1. upon each refresh
    1. once a minute
    1. whenever the user presses the "donate" button and the asset he is donating is ETH

## use cases

### connect web3 wallet (ethereum mainnet) - no donations yet

A user comes to our donation web site and connects his metamask wallet that is switched to the ethereum mainnet:
1. get the wallet address
1. call the `delphi` backend to get the user data associated with the wallet address
1. if the user has not donated anything the backend will return an empty `json` dict.

### connect web3 wallet (ethereum mainnet) - with donation data
A user comes to our donation web site and connects his metamask wallet that is switched to the ethereum mainnet:
1. get the wallet address
1. call the `delphi` backend to get the user data associated with the wallet address
1. the data returned by the `delphi` backend should look similar to this:

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
        },
        {
          "amount": "0.9876540000",
          "asset": "eth",
          "price": "0.00100",
          "status": "confirmed",
          "tokens": "1586905",
          "ts": "2023-10-21T07:56:12Z",
          "tx_hash": "0x7ddb66c1be4ec06e0edd6c2f4ad8b878b2fb5b8075157f5881f0a7f35c5c7cb4",
          "usd_amount": "1586.90"
        }
      ],
      "user_data": {
        "reward": "0",
        "staked": "0",
        "status": "none",
        "tokens": "3568319",
        "total": "3568.31",
        "ts": "2023-10-21T08:28:38.744069Z"
      }
    }
    ```

    The first `n-1` stanzas are the donations made by the user. The last stanza is a summary.

1. display the data above to the user
1. if he has not staked any tokens allow him to do so
1. if he _has_ staked tokens already allow him to unstake them (takes 7 days from the time the unstaking was requested)

### donate

Display the token price and allow the user to
1. select the asset (ETH, USDT or USDC)
1. specify the amount he wants to donate

Whenever the amount changes: display the amount of tokens corresponding to the donation amount specified.

If the user is donating ETH the token amount needs to be calculated as follows:

    ceiling(DA * EP / TP)

where
- DA = donation amount in ETH
- EP = ethereum price in USD
- TP = token price

If the user is donating in stable coin the formula is simpler:

    ceiling(DA / TP)

After the user presses the "donate" button: construct the ethereum transaction and [send it](https://docs.metamask.io/wallet/how-to/send-transactions/) to the network.

If the user came via an affiliate code - call the `delphi` backend and pass the `txHash` and the affiliate code to it.

### stake funds

### unstake funds
