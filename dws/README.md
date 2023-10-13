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

  - Private environment variables, use on the backend side
    - `API_URL`: Specify real backend APIs host
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
