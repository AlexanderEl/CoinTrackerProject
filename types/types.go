package types

import (
	"encoding/json"
)

type AddressData struct {
	Address string
	Balance int64
	NumTxs  int
	Txs     []Transaction
}

type AddressError struct {
	Address string
	Err     error
}

// Transaction data we want to store
type Transaction struct {
	Index int64
	Hash  string
	Data  string
}

// The address data received from the BlockChain API
type RawAddressResponse struct {
	Hash160       string            `json:"hash160"`
	Address       string            `json:"address"`
	NTx           int               `json:"n_tx"`
	NUnredeemed   int               `json:"n_unredeemed"`
	TotalReceived int64             `json:"total_received"`
	TotalSent     int64             `json:"total_sent"`
	FinalBalance  int64             `json:"final_balance"`
	Txs           []json.RawMessage `json:"txs"`
}

type HttpAddressRequestBody struct {
	Address string `json:"address"`
}

type HttpDataResponse struct {
	Address      string `json:"address"`
	Balance      int64  `json:"balance"`
	Transactions []byte `json:"transactions"`
}
