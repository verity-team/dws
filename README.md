# truth memes donation web site

## general

We want to build a 1-page web site similar to https://wallstmemes.com/ -- the donation widget needs to be on top of the page. It shall allow a user to donate ETH, USDT or USDC.

The backend used by the donation web site frontend is called `delphi` and it will expose a [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) to the frontend.

## how to run the dws backend

The dws backend consists of a `postgres` database and 4 services
- `buck`: ETH/latest crawler, checks the latest blocks for donation transactions and inserts the into the database (state: `unconfirmed`)
- `buck`: ETH/finalized crawler, check the finalized blocks for donation transactions and confrms them, also updates the donation campaign statistics and the token price (if/as needed)
- `pulitzer`: pulls the ETH price from 6 exchanges and inserts an average price into the database every minute
- `delphi`: [REST API](https://app.swaggerhub.com/apis/MUHAREM_1/delphi/) server -- only serves data from the database

The backend services are written in `go` -- you will thus need go on your development system. For testing purposes the `postgres` database can be run in a docker container i.e. you will need docker as well.

Once you have these in place you can simply run
1. `go mod tidy` to fetch the dependencies
1. `make` to build the services
1. `make run_db` to run a fresh database in a docker container
1. `bin/pulitzer 2>&1 | tee /tmp/pulitzer-`date +'%Y-%m-%d_%H-%M-%S'`.log` in a separate terminal
1. `bin/delphi | tee /tmp/delphi-`date +'%Y-%m-%d_%H-%M-%S'`.log` in a separate terminal
1. `bin/buck --monitor-latest 2>&1 | tee /tmp/buck-latest-`date +'%Y-%m-%d_%H-%M-%S'`.log`
1. `bin/buck --monitor-final 2>&1 | tee /tmp/buck-final-`date +'%Y-%m-%d_%H-%M-%S'`.log`

All services support the `-p` command-line flag allowing you to set the port they are listening to. The default ports are as follows


|service          | port   | content served |
|:-----------------:|------:|--------------------------------------------|
|delphi      | 8080  | REST API + /live and /ready healtcheck endpoints |
|pulitzer      | 8082  | /live and /ready healtcheck endpoints |
|buck/latest      | 8082  | /live and /ready healtcheck endpoints |
|buck/final      | 8083  | /live and /ready healtcheck endpoints |

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
          "amount": "1.23",
          "asset": "ETH",
          "usd_amount": "2009.82",
          "tokens": "980000",
          "price": "0.002",
          "ts": "2023-10-07T12:48:28+00:00"
        },
        {
          "amount": "760",
          "asset": "USDT",
          "tokens": "380000",
          "price": "0.002",
          "ts": "2023-10-06T12:52:10+00:00"
        }
      ],
      "stats": {
        "total": "1360000",
        "staked": "0",
        "rewards": "0"
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
