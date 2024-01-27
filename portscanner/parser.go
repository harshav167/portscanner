package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open the CSV file
	file, err := os.Open("service-names-port-numbers.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Parse the CSV data
	r := csv.NewReader(file)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// Open the SQLite database
	db, err := sql.Open("sqlite3", "./services.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	createTableSQL := `CREATE TABLE IF NOT EXISTS services (
		"ServiceName" TEXT,
		"PortNumber" TEXT,
		"TransportProtocol" TEXT
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	// Skip the header row and insert data into the SQLite database
	for _, record := range records[1:] {
		serviceName := record[0]
		portNumber := record[1]
		transportProtocol := record[2]

		// Skip rows with empty service name or port number
		if serviceName == "" || portNumber == "" {
			continue
		}

		insertSQL := `INSERT INTO services (ServiceName, PortNumber, TransportProtocol) VALUES (?, ?, ?)`
		statement, err := db.Prepare(insertSQL)
		if err != nil {
			log.Fatal(err)
		}
		defer statement.Close()

		_, err = statement.Exec(serviceName, portNumber, transportProtocol)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Data imported into SQLite successfully.")
}
