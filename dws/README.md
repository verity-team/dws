## Prerequisite

- [Node.js 20.10.0 (LTS)](https://nodejs.org/en)

- Install dependencies

```bash
npm install
```

## Getting started

### For the development server:

```bash
npm run dev
```

### For the production server

- With Docker

```bash
# Build the Docker image
docker build -t dws .

# Run the Docker image
docker run -p 3000:3000 --name dws dws
```

- Without Docker

```bash
# Build Next.js application
npm run build

# Start the output with Node.js
node .next/standalone/server.js
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

## Environment variable setup

- Copy `.env.example` and rename the copied file to `.env` or `.env.local`

- About environment variables

  - Public environment variables, use on the client side

    - `NEXT_PUBLIC_DONATE_PUBKEY`: A wallet address that will receive donations
    - `NEXT_PUBLIC_MIN_ETH`: Minimum value for ETH token donations
    - `NEXT_PUBLIC_MIN_USDT`: Minimum value for USDT token donations (WIP)
    - `NEXT_PUBLIC_API_TIMEOUT`: Maximum waiting time for a response (before the request get cancelled)
    - `NEXT_PUBLIC_LAUNCH_TIME`: Timestamp of launch date in milliseconds
    - `NEXT_PUBLIC_UDATA_REFRESH_TIME_DEFAULT`: Default refresh interval for user's donation data
    - `NEXT_PUBLIC_UDATA_REFRESH_TIME_UNCONFIRMED`: Refresh interval for user's donation data when there are any `unconfirmed` transactions
    - `NEXT_PUBLIC_EMAIL_API_URL`: Host URL for email subscription. Web application will request to `{hostURL}/subscribe.php`
    - `NEXT_PUBLIC_TARGET_NETWORK_ID`: Target chain' id (what network should the web application operate on), hex-encoded. Example: 0x1 for Mainnet, 0xaa36a7 for Sepolia Testnet

      More info at [MetaMask Docs](https://docs.metamask.io/wallet/how-to/connect/detect-network/#chain-ids) or [Chainlist](https://chainid.network/)

    - `NEXT_PUBLIC_TARGET_NETWORK_ALIAS`: Alias of the target network so we can generate link from the tx hash. If use on mainnet, we can ignore this parameter, or set it to an empty string

    - `NEXT_PUBLIC_WC_PROJECT_ID`: Project ID from WalletConnect. This is required to access WalletConnect Explorer and ensure its functionalities on our web application

      Get your project id at [WalletConnect Cloud](https://cloud.walletconnect.com/app) or use this mock project id `cc64c60e9acdab0e270dd3bd452b7fd9`

    - `NEXT_PUBLIC_TARGET_NETWORK_RPC`: Target chain's RPC endpoint. Example: `https://rpc.sepolia.org` for Sepolia Testnet

    - `NEXT_PUBLIC_HOST_URL`: The frontend root URL

    - `NEXT_PUBLIC_FILE_MAX`: Maximum size for file upload (for meme upload feature)

    - `NEXT_PUBLIC_TARGET_SALE=10500000`: Target amount of the token sale in dollars

  - Public environment variables for Galactica server

    - `NEXT_PUBLIC_GALACTICA_API_URL`: Galactica API server's URL

  - Private environment variables, use on the backend side

    - `DWS_API_URL`: Donation Website API server's URL

    - `HOST_URL`: The frontend root URL

      - This is needed to show Twitter card when sharing to Twitter. More about [Twitter Card](https://developer.twitter.com/en/docs/twitter-for-websites/cards/overview/abouts-cards)

    - `API_TIMEOUT`: Specify API request default timeout (in ms)
