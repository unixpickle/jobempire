package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		dieUsage()
	}

	switch os.Args[1] {
	case "master":
		if len(os.Args) != 7 {
			dieUsage()
		}
		slavePort, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid slave port:", os.Args[2])
			os.Exit(1)
		}
		adminPort, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid admin port:", os.Args[3])
			os.Exit(1)
		}
		slavePass := os.Args[4]
		adminPass := os.Args[5]
		configPath := os.Args[6]
		MasterMain(slavePort, adminPort, slavePass, adminPass, configPath)
	case "slave":
		if len(os.Args) != 5 {
			dieUsage()
		}
		host := os.Args[2]
		port, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid master port:", os.Args[2])
			os.Exit(1)
		}
		password := os.Args[4]
		SlaveMain(host, port, password)
	default:
		dieUsage()
	}
}

func dieUsage() {
	fmt.Fprintln(os.Stderr, "Usage: jobempire master <slave_port> <admin_port> <slave_pass>")
	fmt.Fprintln(os.Stderr, "                        <admin_pass> <jobs.json>")
	fmt.Fprintln(os.Stderr, "       jobempire slave <host> <port> <password>")
	fmt.Fprintln(os.Stderr, "\nOptional environment variables:")
	fmt.Fprintln(os.Stderr, " JOB_MEM_LIMIT   maximum memory in MiB (for slave)")
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
