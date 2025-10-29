package data

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/AlexanderEl/cointracker/types"
	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
	Initialize()                                // Initialize the database connnection
	Close()                                     // Close the connection
	StartTransaction()                          // Launch a transaction
	CommitTransaction()                         // Commit the transaction changes
	CheckAddress(address string)                // Check if address exists
	AddAddress(address string)                  // Insert new address
	RemoveAddress(address string)               // Remove existing address
	UpdateBalance(address string, balance int)  // Update existing balance for this address
	UpdateTransactions(data *types.AddressData) // Update transactions for this address
	GetBalance(address string)                  // Retrieve existing balance for address
	GetAllAddresses()                           // Retrieve full list of saved addresses
	GetTransactions(address string)             // Retreive transactions for this address
}

type DatabaseConnection struct {
	conn *sql.DB
}

type WalletData struct {
	Addresss  string
	Balance   int64
	IsDeleted int
	UpdatedAt time.Time
}

// List of all queries in one place
var (
	checkAddressQuery    = "SELECT EXISTS(SELECT 1 FROM wallet WHERE address = ?)"
	insertAddressQuery   = "INSERT INTO wallet (address) VALUES (?)"
	removeAddressQuery   = "DELETE FROM wallet WHERE address = ?"
	removeTxQuery        = "DELETE FROM transactions WHERE address = ?"
	updateBalaceQuery    = "UPDATE wallet SET balance = ? WHERE address = ?"
	updateTxQuery        = "INSERT OR IGNORE INTO transactions (tx_index, hash, address, data) VALUES (?, ?, ?, ?)"
	getBalanceQuery      = "SELECT balance FROM wallet WHERE address = ?"
	getAllAddressesQuery = "SELECT address FROM wallet"
	getTransactionsQuery = "SELECT data FROM transactions WHERE address = ? ORDER BY tx_index DESC"
)

func Initialize() (*DatabaseConnection, error) {
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	initializeTables := `
		CREATE TABLE IF NOT EXISTS wallet (
			address TEXT PRIMARY KEY,
			balance INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS transactions (
			tx_index TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			address TEXT NOT NULL,
			data TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_address ON transactions(address);
		CREATE INDEX IF NOT EXISTS idx_tx_index ON transactions(tx_index DESC);
	`
	_, err = db.Exec(initializeTables)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &DatabaseConnection{
		conn: db,
	}, nil
}

func (db *DatabaseConnection) Close() error {
	if err := db.conn.Close(); err != nil {
		return fmt.Errorf("failed to close database connection with error: %w", err)
	}
	return nil
}

func (db *DatabaseConnection) StartTransaction() (*sql.Tx, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

func (db *DatabaseConnection) CommitTransaction(tx *sql.Tx) error {
	return tx.Commit()
}

func (db *DatabaseConnection) CheckAddress(address string) (bool, error) {
	var exists bool
	if err := db.conn.QueryRow(checkAddressQuery, address).Scan(&exists); err != nil {
		return false, fmt.Errorf("error while checking address existance: %w", err)
	}
	return exists, nil
}

func (db *DatabaseConnection) AddAddress(address string) error {
	result, err := db.conn.Exec(insertAddressQuery, address)
	if err != nil {
		return fmt.Errorf("failure to insert new address: %w", err)
	}
	err = validateEffectedRows(&result, "insertion")
	if err != nil {
		return err
	}

	return nil
}

func (db *DatabaseConnection) RemoveAddress(address string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failure to initalize transaction for removal: %w", err)
	}
	defer tx.Rollback()

	// Removal from wallet table
	result, err := tx.Exec(removeAddressQuery, address)
	if err != nil {
		return fmt.Errorf("failure to remove address: %w", err)
	}
	err = validateEffectedRows(&result, "address removal")
	if err != nil {
		return err
	}

	// Removal from transaction table
	_, err = tx.Exec(removeTxQuery, address)
	if err != nil {
		return fmt.Errorf("failure to remove transactions: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing the removal transaction: %w", err)
	}
	return nil
}

func (db *DatabaseConnection) UpdateBalance(tx *sql.Tx, address string, balance int64) error {
	result, err := tx.Exec(updateBalaceQuery, balance, address)
	if err != nil {
		return fmt.Errorf("failed to update address balance: %w", err)
	}
	err = validateEffectedRows(&result, "balance update")
	if err != nil {
		return err
	}

	return nil
}

func (db *DatabaseConnection) UpdateTransactions(tx *sql.Tx, data *types.AddressData) error {
	for _, t := range data.Txs {
		_, err := tx.Exec(updateTxQuery, t.Index, t.Hash, data.Address, t.Data)

		if err != nil {
			return fmt.Errorf("failed to update transactions for index %d: %w", t.Index, err)
		}
	}

	return nil
}

func (db *DatabaseConnection) GetBalance(address string) (int64, error) {
	row := db.conn.QueryRow(getBalanceQuery, address)
	var balance int64
	err := row.Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("failed to get address balance: %w", err)
	}

	return balance, nil
}

func (db *DatabaseConnection) GetAllAddresses() (*[]string, error) {
	rows, err := db.conn.Query(getAllAddressesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve addresses: %w", err)
	}
	defer rows.Close()

	var data []string
	for rows.Next() {
		var address string
		if err := rows.Scan(&address); err != nil {
			return nil, fmt.Errorf("failed to read in all addresses: %w", err)
		}
		data = append(data, address)
	}
	return &data, nil
}

func (db *DatabaseConnection) GetTransactions(address string) (*[]string, error) {
	rows, err := db.conn.Query(getTransactionsQuery, address)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve transactions: %w", err)
	}
	defer rows.Close()

	var txs []string
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to read transactions: %w", err)
		}
		txs = append(txs, data)
	}
	return &txs, nil
}

func validateEffectedRows(result *sql.Result, operation string) error {
	numRows, err := (*result).RowsAffected()
	if err != nil {
		return fmt.Errorf("error while getting number of effected rows in %s: %w", operation, err)
	}

	if numRows != 1 {
		return fmt.Errorf("invalid number of effected rows in %s - Expected: 1, Actual: %d", operation, numRows)
	}

	return nil
}
