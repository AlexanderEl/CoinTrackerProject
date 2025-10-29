# CoinTracker Wallet Manager Project
A little project for my application to CoinTracker

## Overview
Here is a simple web service API that scrapes publicly available Bitcoin blockchain data for the provided addresses then stores the transaction history and the latest balance.


# Project Breakdown
The service launches on `localhost:9000`

## API Endpoints

1. Add new address: `POST` `localhost:9000/address?address=<BTC address here>`

Add a new Bitcoin address to the wallet

2. Remove existing address: `DELTE` `localhost:9000/address?address=<BTC address here>`

Remove an existing address from the wallet

3. Initialize new data synchronization: `POST` `localhost:9000/sync`

Kick off a data sync

4. Get wallet data: `POST` `localhost:9000/data`

Retrieve all the stored wallet data


# How to run it
Instead the SQLite library for Go by executing 
```
go get github.com/mattn/go-sqlite3
```

With the only dependency installed, now navigate to the runner directory and launch the service by executing
```
go run main.go
```

Once the service is started, make API calls to interact with it

## Future Expanse Ideas
- **Created a smart data sync scheduler**: to continously pull data without being rate limited
- **Use the multi address API endpoint for optimizing fetching of latest transactions**: right now data is fetched by getting all transactions for each of the addresses rather than getting the latest transactions for all fo the addresses
- **Optimize fetching to Bitcoin block mining**: we should never fetch data before a new block is mined as no new data is available

# Project Assumptions
1. This would be use as small scale (< 100 Bitcoin addresses) - realistically < 10
2. This would be individually run (locally hosted for personal addresses)
3. High surface aea for increasing paralliziation of workflows if API was not rate limiting

## License
This package is licensed under the MIT License - see the LICENSE file for details.
