package leaseweb

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/rid/kube-vip-leaseweb/pkg/kubevip"
	"github.com/rid/leasewebgo"
	log "github.com/sirupsen/logrus"
)

type ErrorMessage struct {
	CorrelationID string `json:"correlationId"`
	ErrorCode     string `json:"errorCode"`
	ErrorMessage  string `json:"errorMessage"`
}

// AttachFIP will use the packet APIs to move an FIP and attach to a host
func AttachFIP(c *leasewebgo.Client, k *kubevip.Config, vip string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	// Get our IP from our hostname (should be second subdomain)
	// ex: dedi1-node1.**23-106-60-155**.lon-01.uk.appsolo.com
	// NOTE: Change this to reverse DNS lookup if you don't want to put IP in host (although this is faster)
	localIP := strings.Join(strings.Split(strings.Split(hostname, ".")[1], "-"), ".")

	log.Infof("Attaching FIP address [%s] to host IP address [%s]", vip, localIP)

	// Prefer Address over VIP
	if vip == "" {
		vip = k.Address
		if vip == "" {
			vip = k.VIP
		}
	}

	// get the rangeId for the floating IP
	floatingIPRange, err := findRange(vip, c)
	if err != nil {
		return err
	}

	// Assign the floating IP to this device
	_, _, body, err := c.FloatingIps.UpdateRange(floatingIPRange.Id, strings.Join([]string{vip, "32"}, "_"), &leasewebgo.RangeUpdateRequest{AnchorIp: localIP}, nil)
	if err != nil {
		if len(body) > 0 {
			errorMessage := &ErrorMessage{}
			err = json.Unmarshal(body, errorMessage)
			if err != nil {
				return err
			}
			if errorMessage.ErrorCode == "404" {
				if strings.Contains(errorMessage.ErrorMessage, "does not exist") {
					_, _, _, err = c.FloatingIps.CreateRange(floatingIPRange.Id, &leasewebgo.RangeCreateRequest{FloatingIp: vip, AnchorIp: localIP}, nil)
					if err != nil {
						return err
					}
					log.Infof("Attached FIP address [%s] to host IP address [%s]", vip, localIP)
				}
			}
		}
	} else {
		log.Infof("Attached FIP address [%s] to host IP address [%s]", vip, localIP)
	}

	return nil
}
