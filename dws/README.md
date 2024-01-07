## Prerequisite

- [Node 18 LTS](https://nodejs.org/en/download)
- Yarn Classic (v1.22)

- Notes: You can install the latest yarn and switch the version back to classic with

```
yarn set version classic
```

## Setup

- Copy `.env.example` and rename the copy to `.env` or `.env.local`

- About environment variables

  - Public environment variables, use on the client side

    - `NEXT_PUBLIC_DONATE_PUBKEY`: A wallet address that will receive donations
    - `NEXT_PUBLIC_REWARD_PRICE`: Fixed price for reward coin (People will receive reward coins for making donations)
    - `NEXT_PUBLIC_MIN_ETH`: Minimum value for ETH token donations
    - `NEXT_PUBLIC_MIN_USDT`: Minimum value for USDT token donations (WIP)
    - `NEXT_PUBLIC_API_TIMEOUT`: Maximum waiting time for a response (before the request get cancelled)
    - `NEXT_PUBLIC_LAUNCH_TIME`: Timestamp of launch date in milliseconds
    - `NEXT_PUBLIC_UDATA_REFRESH_TIME_DEFAULT`: Default refresh interval for user's donation data
    - `NEXT_PUBLIC_UDATA_REFRESH_TIME_UNCONFIRMED`: Refresh interval for user's donation data when there are any `unconfirmed` transactions
    - `NEXT_PUBLIC_EMAIL_API_URL`: Host URL for email subscription. Web application will request to `{hostURL}/subscribe.php`
    - `NEXT_PUBLIC_TARGET_NETWORK_ID`: Target chain' id (what network should the web application operate on), hex-encoded. Example: 0x1 for Mainnet, 0xaa36a7 for Sepolia Testnet

      More info at [MetaMask Docs](https://docs.metamask.io/wallet/how-to/connect/detect-network/#chain-ids) or [Chainlist](https://chainid.network/)

    - `NEXT_PUBLIC_WC_PROJECT_ID`: Project ID from WalletConnect. This is required to access WalletConnect Explorer and ensure its functionalities on our web application

      Get your project id at [WalletConnect Cloud](https://cloud.walletconnect.com/app) or use this mock project id `cc64c60e9acdab0e270dd3bd452b7fd9`

    - `NEXT_PUBLIC_TARGET_NETWORK_RPC`: Target chain's RPC endpoint. Example: `https://rpc.sepolia.org` for Sepolia Testnet

    - `NEXT_PUBLIC_HOST_URL`: The frontend root URL

    - `NEXT_PUBLIC_FILE_MAX`: Maximum size for file upload (for meme upload feature)

  - Public environment variables for Galactica server

    - `NEXT_PUBLIC_GALACTICA_API_URL`: Galactica API server's URL

  - Private environment variables, use on the backend side

    - `DWS_API_URL`: Donation Website API server's URL

    - `HOST_URL`: The frontend root URL

      - This is needed to show Twitter card when sharing to Twitter. More about [Twitter Card](https://developer.twitter.com/en/docs/twitter-for-websites/cards/overview/abouts-cards)

    - `API_TIMEOUT`: Specify API request default timeout (in ms)

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

Connect the website with your Metamask wallet to start making donations

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/deployment) for more details.
