package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"
)

const (
	MAX = 65535
)

type model struct {
	address   string
	startPort int
	endPort   int
	openPorts map[int]string // Change to map for port-service mapping
	finished  bool
}

func initialModel(address string, startPort, endPort int) model {
	return model{
		address:   address,
		startPort: startPort,
		endPort:   endPort,
		openPorts: make(map[int]string),
		finished:  false,
	}
}

type portScannedMsg struct {
	port    int
	service string
}

type finishMsg struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case portScannedMsg:
		m.openPorts[msg.port] = msg.service // Assign the service to the port

	case finishMsg:
		m.finished = true
		// sort.Ints(m.openPorts)
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	if m.finished {
		var ports []int
		for port := range m.openPorts {
			ports = append(ports, port)
		}
		sort.Ints(ports) // Sort the slice of port numbers

		var result string
		for _, port := range ports {
			service := m.openPorts[port]
			if service != "" {
				result += fmt.Sprintf("%d is open (service: %s)\n", port, service)
			} else {
				result += fmt.Sprintf("%d is open\n", port)
			}
		}
		return result
	}
	return "Scanning... Press q to quit\n"
}

func getService(port int) string {
	db, err := sql.Open("sqlite3", "./services.db")
	if err != nil {
		fmt.Println("Error opening database:", err)
		return ""
	}
	defer db.Close()

	var serviceName string
	query := `SELECT ServiceName FROM services WHERE PortNumber = ?`
	err = db.QueryRow(query, port).Scan(&serviceName)
	if err != nil {
		if err != sql.ErrNoRows {
			fmt.Println("Error querying database:", err)
		}
		return ""
	}

	return serviceName
}
func scanPort(port int, address string, wg *sync.WaitGroup, p chan<- portScannedMsg) {
	defer wg.Done()

	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, port), timeout)
	if err == nil {
		service := getService(port)
		p <- portScannedMsg{port: port, service: service}
		conn.Close()
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./port <IP_ADDRESS>")
		os.Exit(1)
	}
	address := os.Args[1]
	startPort := 1
	endPort := MAX

	m := initialModel(address, startPort, endPort)
	p := make(chan portScannedMsg)
	var wg sync.WaitGroup

	for i := startPort; i <= endPort; i++ {
		wg.Add(1)
		go scanPort(i, address, &wg, p)
	}

	go func() {
		wg.Wait()
		close(p)
	}()

	program := tea.NewProgram(m)

	go func() {
		for msg := range p {
			program.Send(msg)
		}
		program.Send(finishMsg{})
	}()

	if err := program.Start(); err != nil {
		fmt.Printf("Could not start the program: %v", err)
	}
}
