package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/AlexanderEl/cointracker"
	"github.com/AlexanderEl/cointracker/types"
)

var wm *cointracker.Wallet

func main() {
	manager, err := cointracker.CreateWalletManager()
	if err != nil {
		fmt.Println("Exiting due to failure to initialize wallet manager with error:", err)
		return
	}
	wm = manager

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		msg := `Welcome to my CoinTracker Demo!

Try out these endpoints:

POST	localhost:9000/address 	- Add a new address
DELETE	localhost:9000/address 	- Remove existing address
GET	localhost:9000/sync	- Synchronize the data from Blockchain
GET	localhost:9000/data	- Get all wallet data`
		fmt.Fprintf(w, "%s", msg)
	})

	http.HandleFunc("/address", updateAddress)
	http.HandleFunc("/sync", syncWallets)
	http.HandleFunc("/data", retrieveWalletData)

	fmt.Println("Server launched: localhost:9000")
	http.ListenAndServe(":9000", nil)
}

func updateAddress(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")

	switch r.Method {
	case "POST":
		addAddress(w, address)
	case "DELETE":
		removeAddress(w, address)
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func addAddress(w http.ResponseWriter, address string) {
	if exists, err := wm.AddAddress(address); err != nil {
		log.Println("Error while adding new address:", err)
		http.Error(w, "Error adding new address", http.StatusInternalServerError)
	} else if exists {
		log.Println("Provided addresss already exist")
		http.Error(w, "Cannot add duplicate addresses", http.StatusConflict)
	}

}

func removeAddress(w http.ResponseWriter, address string) {
	if exists, err := wm.RemoveAddress(address); err != nil {
		log.Println("Error while removing given address:", err)
		http.Error(w, "Error removing given address", http.StatusInternalServerError)
	} else if !exists {
		log.Println("Provided addresss does not exist")
		http.Error(w, "Cannot remove non-existent address", http.StatusNotFound)
	}
}

func syncWallets(w http.ResponseWriter, r *http.Request) {
	go wm.SyncWallets() // Launch data sync in background

	w.Write([]byte("Wallet Sync Initiated"))
}

func retrieveWalletData(w http.ResponseWriter, r *http.Request) {
	addresses, err := wm.GetAllAddresses()
	if err != nil {
		log.Println("Error while retriving wallet data:", err)
		http.Error(w, "Error retrieving all addresses", http.StatusInternalServerError)
		return
	}

	responseAddressList := make([]types.HttpDataResponse, len(*addresses))
	for index, address := range *addresses {
		balance, err := wm.GetBalance(address)
		if err != nil {
			log.Println("Error while retriving address balance:", err)
			http.Error(w, "Error retrieving address balance", http.StatusInternalServerError)
			return
		}

		txList, err := wm.GetTransactions(address)
		if err != nil {
			log.Println("Error while retriving address transactions:", err)
			http.Error(w, "Error retrieving all transactions", http.StatusInternalServerError)
			return
		}

		responseAddressList[index] = types.HttpDataResponse{
			Address:      address,
			Balance:      balance,
			Transactions: txList,
		}
	}

	output, err := json.Marshal(responseAddressList)
	if err != nil {
		log.Println("Error while encoding all data:", err)
		http.Error(w, "Error retriving all data", http.StatusInternalServerError)
		return
	}

	w.Write(output)
}
