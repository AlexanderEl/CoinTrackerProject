package cointracker

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/AlexanderEl/cointracker/data"
	"github.com/AlexanderEl/cointracker/types"
)

// WalletManager handles wallet synchronization and storage
type WalletManager interface {
	CreateWalletManager()           // Initialize the wallet manager
	AddAddress(address string)      // Add new address to wallet
	RemoveAddress(address string)   // Remove address from wallet
	SyncWallets()                   // Update the wallet data with fresh data
	GetBalance(address string)      // Retrieve the balance for this address
	GetTransactions(address string) // Retrieve transactions for this address
	GetAllAddresses()               // Retrieve all addresses in wallet
}

type Wallet struct {
	data *data.DatabaseConnection
}

func CreateWalletManager() (*Wallet, error) {
	conn, err := data.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database connection: %w", err)
	}

	return &Wallet{
		data: conn,
	}, nil
}

func (w *Wallet) AddAddress(address string) (bool, error) {
	if len(address) == 0 || len(address) > 42 {
		return false, fmt.Errorf("cannot add address of invalid length")
	}

	exists, err := w.data.CheckAddress(address)
	if err != nil {
		return false, err
	}
	if exists {
		return exists, nil
	}

	if err = w.data.AddAddress(address); err != nil {
		return false, fmt.Errorf("failure to add new address")
	}

	// Launch a data fetch for new address in the background
	go w.fetchAddressData(address)

	return false, nil
}

func (w *Wallet) RemoveAddress(address string) (bool, error) {
	exists, err := w.data.CheckAddress(address)
	if err != nil {
		return false, err
	}
	if !exists {
		return exists, nil
	}

	return exists, w.data.RemoveAddress(address)
}

func (w *Wallet) SyncWallets() {
	addresses, err := w.data.GetAllAddresses()
	if err != nil {
		log.Printf("failed to get all addresses for wallet sync: %s", err)
	}

	results := make([]*types.AddressData, len(*addresses))
	for _, address := range *addresses {
		if data, err := w.fetchAddressData(address); err != nil {
			log.Printf("Bad fetch for address: %s with error %s\n", address, err)
		} else {
			results = append(results, data)
		}

		time.Sleep(time.Second * 30) // Cool down timer to avoid getting rate limited
	}

	// Start transaction to insert values
	err = w.data.StartTransaction()
	if err != nil {
		log.Printf("failed to begin transaction during sync: %s", err)
	}

	for _, result := range results {
		if err := w.data.UpdateBalance(result.Address, result.Balance); err != nil {
			log.Println("Failed to update the balance for address: ", result.Address)
			log.Printf("failed to update balance for address: %s with error: %s", result.Address, err)
		}
		if err := w.data.UpdateTransactions(result); err != nil {
			log.Println("Failed to update the transactions for address: ", result.Address)
			log.Printf("failed to update balance for address: %s with error: %s", result.Address, err)
		}
	}

	if err := w.data.CommitTransaction(); err != nil {
		log.Println("Failed to commit the transaction for setting fresh values")
		log.Printf("failed to commit transaction: %s", err)
	}

	log.Println("Successfully updated all values for wallet")
}

func (w *Wallet) GetBalance(address string) (int64, error) {
	return w.data.GetBalance(address)
}

func (w *Wallet) GetTransactions(address string) (*[]byte, error) {
	txs, err := w.data.GetTransactions(address)
	if err != nil {
		return nil, fmt.Errorf("error while getting transactions: %w", err)
	}

	// Iterate over JSON transaction rows and combine into a single list
	var txList []any
	for _, txData := range *txs {
		var data any
		if err := json.Unmarshal([]byte(txData), &data); err != nil {
			return nil, fmt.Errorf("failure parsing transaction data: %w", err)
		}
		txList = append(txList, data)
	}

	// Convert the transactions list into a single JSON byte array
	bytes, err := json.Marshal(txList)
	if err != nil {
		return nil, fmt.Errorf("error encoding transactions into JSON: %w", err)
	}
	return &bytes, nil
}

func (w *Wallet) GetAllAddresses() (*[]string, error) {
	return w.data.GetAllAddresses()
}

// Retrieve address data from API
func (w *Wallet) fetchAddressData(address string) (*types.AddressData, error) {
	limit, offset := 50, 0
	baseURL := fmt.Sprintf("https://blockchain.info/rawaddr/%s?limit=%d&cors=true", address, limit)

	var data *types.AddressData = nil
	var allTxs []types.Transaction

	for {
		url := fmt.Sprintf("%s&offset=%d", baseURL, offset)
		resp, err := http.Get(url)
		if err != nil {
			log.Println("Error while making request to API:", err)
			return nil, fmt.Errorf("failed to fetch address data: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Println("Bad response code from API call:", resp.StatusCode)
			return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error while reading response body:", err)
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var response types.RawAddressResponse
		if err := json.Unmarshal(body, &response); err != nil {
			log.Println("Error parsing JSON body", err)
			return nil, fmt.Errorf("failed to parse API resonse body into raw response: %w", err)
		}

		// Only set once on first pass
		if data == nil {
			data = &types.AddressData{
				Address: response.Address,
				NumTxs:  response.NTx,
				Balance: response.FinalBalance,
			}
		}

		// Process transactions in batches and using partial parsing
		for _, rawTx := range response.Txs {
			var txMap map[string]any
			if err := json.Unmarshal(rawTx, &txMap); err != nil {
				log.Println("Error parsing JSON transactions:", err)
				return nil, fmt.Errorf("error while parsing transactions: %w", err)
			}

			allTxs = append(allTxs, types.Transaction{
				Index: int64(txMap["tx_index"].(float64)),
				Hash:  txMap["hash"].(string),
				Data:  string(rawTx),
			})
		}

		// Check if more pages exist
		offset += limit
		if len(response.Txs) < limit || offset >= response.NTx {
			break
		}

		time.Sleep(time.Second * 15) // To avoid rate limiting based on their requirements: https://www.blockchain.com/explorer/api/q
	}

	data.Txs = allTxs // Set the parsed transactions
	return data, nil
}
