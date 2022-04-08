package leaseweb

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"

	"github.com/rid/leasewebgo"
	log "github.com/sirupsen/logrus"
)

func findRange(floatingIP string, c *leasewebgo.Client) (*leasewebgo.Range, error) {
	ranges, _, _, err := c.FloatingIps.ListRanges(&leasewebgo.ListOptions{QueryParams: map[string]string{"limit": "50"}})
	if err != nil {
		return nil, err
	}

	totalCount := ranges.Meta.Totalcount
	offset := 0

	for totalCount > 0 {
		for _, ipRange := range ranges.Ranges {
			log.Println("ipRange:", ipRange.Id)
			// convert string to IPNet struct
			_, ipv4Net, err := net.ParseCIDR(ipRange.Range)
			if err != nil {
				return nil, err
			}

			// convert IPNet struct mask and address to uint32
			// network is BigEndian
			mask := binary.BigEndian.Uint32(ipv4Net.Mask)
			start := binary.BigEndian.Uint32(ipv4Net.IP)

			// find the final address
			finish := (start & mask) | (mask ^ 0xffffffff)

			// loop through addresses as uint32
			for i := start; i <= finish; i++ {
				// convert back to net.IP
				ip := make(net.IP, 4)
				binary.BigEndian.PutUint32(ip, i)
				if ip.String() == floatingIP {
					return &ipRange, nil
				}
			}
			totalCount--
			offset++
		}
		ranges, _, _, err = c.FloatingIps.ListRanges(&leasewebgo.ListOptions{QueryParams: map[string]string{"limit": "50", "offset": strconv.Itoa(offset)}})
		if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("failed to find range for floating ip %s", floatingIP)
}

//GetLeasewebConfig will lookup the configuration from a file path
func GetLeasewebConfig(providerConfig string) (string, error) {
	var config struct {
		AuthToken string `json:"apiKey"`
	}
	// get our token
	if providerConfig != "" {
		configBytes, err := ioutil.ReadFile(providerConfig)
		if err != nil {
			return "", fmt.Errorf("failed to get read configuration file at path %s: %v", providerConfig, err)
		}
		err = json.Unmarshal(configBytes, &config)
		if err != nil {
			return "", fmt.Errorf("failed to process json of configuration file at path %s: %v", providerConfig, err)
		}
	}
	return config.AuthToken, nil
}
