package main

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"strings"
	"time"

	"github.com/linode/linodego"
	"golang.org/x/oauth2"

	"log"
	"net/http"
	"os"
)

var (
	httpClient        http.Client
	externalIPAddress string
)

type LinodeDomainClient interface {
	ListDomainRecords(ctx context.Context, domainID int, opts *linodego.ListOptions) ([]linodego.DomainRecord, error)
	ListDomains(ctx context.Context, opts *linodego.ListOptions) ([]linodego.Domain, error)
	CreateDomainRecord(ctx context.Context, domainID int, domainrecord linodego.DomainRecordCreateOptions) (*linodego.DomainRecord, error)
	UpdateDomainRecord(ctx context.Context, domainID int, id int, domainrecord linodego.DomainRecordUpdateOptions) (*linodego.DomainRecord, error)
}

type LinodeProvider struct {
	Client LinodeDomainClient
}

type LinodeDomain struct {
	domainFilter      string
	subdomainFilter   string
	domainID          int
	recordID          int
	externalIPAddress string
	remoteIPAddress   string
}

type appError struct {
	Error   error
	Message string
}

func main() {

	apiKey, ok := os.LookupEnv("LINODE_TOKEN")
	if !ok {
		log.Fatal("Could not find LINODE_TOKEN environment vairable, exiting.")
	}

	hostname, ok := os.LookupEnv("DNS_HOSTNAME")
	if !ok {
		log.Fatal("Could not find DNS_HOSTNAME environment vairable, exiting.")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiKey})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	p := linodego.NewClient(oauth2Client)

	debug, ok := os.LookupEnv("DEBUG")
	if !ok {
		log.Println("Could not find DEBUG environment vairable, assuming false.")
	}
	if debug == "true" {
		p.SetDebug(true)
		log.Println("Debug enabled")
	}

	LinodeProvider := &LinodeProvider{
		Client: &p,
	}

	host := strings.SplitN(hostname, ".", 2)

	LinodeDomain := &LinodeDomain{
		domainFilter:    host[1],
		subdomainFilter: host[0],
	}

	linodeData, err := getLinodeDomain(LinodeProvider, LinodeDomain)
	if err != nil {
		log.Fatal(err)
	}

	externalIPAddress = getExternalIP()[:len(getExternalIP())-1]

	if externalIPAddress == linodeData.remoteIPAddress {
		log.Println("External IP already in sync, skipping update.")
		os.Exit(0)
	} else {
		log.Println("External IP is not in sync with remote IP, updating")

		UpdateOptions := linodego.DomainRecordUpdateOptions{
			Type:   "A",
			Name:   LinodeDomain.subdomainFilter,
			Target: externalIPAddress,
		}

		res, err := p.UpdateDomainRecord(context.Background(), linodeData.domainID, linodeData.recordID, UpdateOptions)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("Domain %s updated successfully to %s", res.Name, res.Target)
		}
	}

}

func getExternalIP() string {

	timeout := time.Duration(5 * time.Second)

	httpClient = http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	request, err := http.NewRequest(http.MethodGet, "http://ifconfig.co", nil)

	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	if b, err := ioutil.ReadAll(response.Body); err == nil {
		return string(b)
	}

	return ""
}

func getLinodeDomain(p *LinodeProvider, hr *LinodeDomain) (*LinodeDomain, *appError) {

	res, err := p.Client.ListDomains(context.Background(), &linodego.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, zone := range res {

		if hr.domainFilter == zone.Domain {
			res, err := p.Client.ListDomainRecords(context.Background(), zone.ID, nil)
			if err != nil {
				log.Fatal(err)
			}

			for _, rec := range res {
				if hr.subdomainFilter == rec.Name {
					log.Printf("Found subdomain: %s with existing IP of: %s", rec.Name, rec.Target)
					LinodeDomain := &LinodeDomain{
						domainID:        zone.ID,
						recordID:        rec.ID,
						remoteIPAddress: rec.Target,
					}

					return LinodeDomain, nil
				}

			}
		}

	}

	return nil, &appError{err, "No domain or subdomain found"}
}
