package applicant

import (
	"encoding/json"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/cloudflare"

	"github.com/usual2970/certimate/internal/domain"
)

type cloudflareApplicant struct {
	option *ApplyOption
}

func NewCloudflareApplicant(option *ApplyOption) Applicant {
	return &cloudflareApplicant{
		option: option,
	}
}

func (a *cloudflareApplicant) Apply() (*Certificate, error) {
	access := &domain.CloudflareAccess{}
	json.Unmarshal([]byte(a.option.Access), access)

	config := cloudflare.NewDefaultConfig()
	config.AuthToken = access.DnsApiToken
	if a.option.Timeout != 0 {
		config.PropagationTimeout = time.Duration(a.option.Timeout) * time.Second
	}

	provider, err := cloudflare.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}

	return apply(a.option, provider)
}
