# truth memes donation web site

## rules
1. all amounts are passed as string and should be decoded to a `decimal` type to preserve precision

## use cases

### connect web3 wallet (ethereum mainnet) - no donation

A user comes to our donation web site and connects his metamask wallet that is switched to the ethereum mainnet:
1. get the wallet address
1. call the `dws` backend to get the user data associated with the wallet address
1. if the user has not donated anything the backend will return an empty `json` dict.

### connect web3 wallet (ethereum mainnet) - with donation data
A user comes to our donation web site and connects his metamask wallet that is switched to the ethereum mainnet:
1. get the wallet address
1. call the `dws` backend to get the user data associated with the wallet address
1. the data returned by the `dws` backend should look similar to this:

    ```json
    {
      "user_data": [
        {
          "amount": "1.23",
          "asset": "ETH",
          "tokens": "980000",
          "price": "0.002",
          "date": "2023-10-07T12:48:28+00:00"
        },
        {
          "amount": "760",
          "asset": "USDT",
          "tokens": "380000",
          "price": "0.002",
          "date": "2023-10-06T12:52:10+00:00"
        },
        {
          "total": "1360000",
          "staked": "28000",
          "date": "2023-10-06T13:04:24+00:00",
          "rewards": "0"
        }
      ]
    }
    ```

    The first `n-1` stanzas are the donations made by the user. The last stanza is a summary.

1. display the data above to the user
1. if he has not staked any tokens allow him to do so
1. if he _has_ staked tokens already allow him to unstake them (takes 7 days from the time the unstaking was requested)

### donate ETH

### donate stable coin (usdt or usdc)

### stake funds

### unstake funds
