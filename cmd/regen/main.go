package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"
)

var (
	project = flag.String("project", "gcping-1369", "Project to use")
	tok     = flag.String("tok", "", "Auth token")
	outFile = flag.String("out", "config.js", "Output file")
)

var tmpl = template.Must(template.New("name").Parse(`
var _URLS = {
{{range .}}  "{{.Region}}": "{{.IP}}/ping",
{{end}}};
`))

func main() {
	flag.Parse()

	if *tok == "" {
		log.Fatal("Must provide -tok")
	}

	of, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("os.Open(%s): %v", *outFile, err)
	}
	defer of.Close()

	addresses := append(computeAddresses(), runAddresses()...)
	sort.Slice(addresses, func(i, j int) bool { return addresses[i].Region < addresses[j].Region })

	if err := tmpl.Execute(io.MultiWriter(os.Stdout, of), addresses); err != nil {
		log.Fatalf("tmpl.Execute: %v", err)
	}
}

type address struct{ Region, IP string }

func runAddresses() []address {
	// Get services.
	url := fmt.Sprintf("https://run.googleapis.com/v1/projects/%s/locations/-/services", *project)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("NewRequest: GET %s: %v", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+*tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var response struct {
		Items []struct {
			Metadata struct {
				Labels struct {
					Location string `json:"cloud.googleapis.com/location"`
				} `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Address struct {
					URL string `json:"url"`
				} `json:"address"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Fatalf("json.Decode: %v", err)
	}

	var addresses []address
	for _, i := range response.Items {
		addresses = append(addresses, address{i.Metadata.Labels.Location + "-cloudrun", i.Status.Address.URL})
	}
	return addresses
}

func computeAddresses() []address {
	// Get instances and IPs.
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/aggregated/addresses", *project)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("NewRequest: GET %s: %v", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+*tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var response struct {
		Items map[string]struct {
			Addresses []struct {
				Address string `json:"address"`
			} `json:"addresses"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Fatalf("json.Decode: %v", err)
	}

	var addresses []address
	for reg, addrs := range response.Items {
		reg = strings.TrimPrefix(reg, "regions/")
		addresses = append(addresses, address{reg, "http://" + addrs.Addresses[0].Address})
	}
	return addresses
}
