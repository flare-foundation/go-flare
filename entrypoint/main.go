package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
}
type rpcResponse struct {
	Result json.RawMessage `json:"result"`
}

func fetchPublicIP() (string, error) {
	resp, err := http.Get("https://flare.network/cdn-cgi/trace")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "ip=") {
			return strings.TrimPrefix(line, "ip="), nil
		}
	}

	return "", fmt.Errorf("no ip= line in trace")
}

func parseIPResult(raw json.RawMessage) (string, error) {
	var obj struct {
		IP json.RawMessage `json:"ip"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		var single string
		if err := json.Unmarshal(obj.IP, &single); err == nil {
			return single, nil
		}
		var arr []string
		if err := json.Unmarshal(obj.IP, &arr); err == nil {
			return strings.Join(arr, ","), nil
		}
		return "", fmt.Errorf("unexpected ip field format: %s", obj.IP)
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return single, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return strings.Join(arr, ","), nil
	}
	return "", fmt.Errorf("unexpected result format: %s", string(raw))
}

func parseNodeIDResult(raw json.RawMessage) (string, error) {
	var obj struct {
		NodeID string `json:"nodeID"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.NodeID, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	return "", fmt.Errorf("unexpected nodeID format: %s", string(raw))
}

func rpcCall(client *http.Client, url, method string) (json.RawMessage, error) {
	body, _ := json.Marshal(rpcRequest{"2.0", 1, method})
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var wrap rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&wrap); err != nil {
		return nil, err
	}
	return wrap.Result, nil
}

func main() {
	if os.Getenv("AUTOCONFIGURE_PUBLIC_IP") == "1" {
		if os.Getenv("PUBLIC_IP") == "" {
			fmt.Fprintln(os.Stderr, "Autoconfiguring public IP")
			ip, err := fetchPublicIP()
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to get ip")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Got public address %s \n", ip)
			os.Setenv("PUBLIC_IP", ip)
		} else {
			msg := fmt.Sprintf(
				`/!\ AUTOCONFIGURE_PUBLIC_IP is enabled, but PUBLIC_IP is already `+
					`set to '%s'! Skipping autoconfigure and using current PUBLIC_IP value!`+"\n",
				os.Getenv("PUBLIC_IP"),
			)
			fmt.Fprint(os.Stderr, msg)
		}
	}

	if os.Getenv("AUTOCONFIGURE_BOOTSTRAP") == "1" {
		endpoints := []string{os.Getenv("AUTOCONFIGURE_BOOTSTRAP_ENDPOINT")}
		if fb := os.Getenv("AUTOCONFIGURE_FALLBACK_ENDPOINTS"); fb != "" {
			for _, e := range strings.Split(fb, ",") {
				e = strings.TrimSpace(e)
				if e != "" {
					endpoints = append(endpoints, e)
				}
			}
		}

		var endpoint string
		fmt.Fprintln(os.Stderr, "Trying provided bootstrap endpoints")
		client := http.Client{Timeout: 5 * time.Second}
		probe := []byte(`{"jsonrpc":"2.0","id":1,"method":"info.getNodeIP"}`)

		for _, ep := range endpoints {
			fmt.Fprintf(os.Stderr, "  Trying endpoint %s\n", ep)

			resp, err := client.Post(ep, "application/json", bytes.NewReader(probe))
			if err != nil {
				fmt.Fprintf(os.Stderr, "    error: %v\n", err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				endpoint = ep
				break
			}
			fmt.Fprintln(os.Stderr, "    Failed! The endpoint is unreachable.")
		}

		if endpoint == "" {
			fmt.Fprintln(os.Stderr, "  None of provided bootstrap endpoints worked!")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "found endpoint : ", endpoint)

		fmt.Fprintln(os.Stderr, "Autoconfiguring bootstrap IPs and IDs")

		rawIPs, err := rpcCall(&client, endpoint, "info.getNodeIP")
		if err != nil {
			fmt.Fprintln(os.Stderr, "  getNodeIP RPC failed:", err)
			os.Exit(1)
		}
		bootstrap_IPs, err := parseIPResult(rawIPs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "  parsing IPs failed:", err)
			os.Exit(1)
		}

		rawIDs, err := rpcCall(&client, endpoint, "info.getNodeID")
		if err != nil {
			fmt.Fprintln(os.Stderr, "  getNodeID RPC failed:", err)
			os.Exit(1)
		}

		bootstrap_IDs, err := parseNodeIDResult(rawIDs)
		if err != nil {
			fmt.Fprintln(os.Stderr, "  parsing IDs failed:", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "  Got bootstrap ips: '%s'\n", bootstrap_IPs)
		fmt.Fprintf(os.Stderr, "  Got bootstrap ids: '%s'\n", bootstrap_IDs)

		os.Setenv("BOOTSTRAP_IPS", bootstrap_IPs)
		os.Setenv("BOOTSTRAP_IDS", bootstrap_IDs)
	}

	args := []string{
		"--http-host", os.Getenv("HTTP_HOST"),
		"--http-port", os.Getenv("HTTP_PORT"),
		"--staking-port", os.Getenv("STAKING_PORT"),
		"--public-ip", os.Getenv("PUBLIC_IP"),
		"--db-dir", os.Getenv("DB_DIR"),
		"--db-type", os.Getenv("DB_TYPE"),
		"--bootstrap-ips", os.Getenv("BOOTSTRAP_IPS"),
		"--bootstrap-ids", os.Getenv("BOOTSTRAP_IDS"),
		"--bootstrap-beacon-connection-timeout", os.Getenv("BOOTSTRAP_BEACON_CONNECTION_TIMEOUT"),
		"--chain-config-dir", os.Getenv("CHAIN_CONFIG_DIR"),
		"--log-dir", os.Getenv("LOG_DIR"),
		"--log-level", os.Getenv("LOG_LEVEL"),
		"--network-id", os.Getenv("NETWORK_ID"),
	}
	if extra := os.Getenv("EXTRA_ARGUMENTS"); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	fmt.Fprintln(os.Stderr, args)
	path := "/app/build/avalanchego"

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "file does not exist")
		os.Exit(1)
	} else {
		cmd := exec.Command(path, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintln(os.Stderr, "failed to start avalanchego:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
