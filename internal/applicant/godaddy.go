package applicant

import (
	"encoding/json"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/godaddy"

	"github.com/usual2970/certimate/internal/domain"
)

type godaddyApplicant struct {
	option *ApplyOption
}

func NewGoDaddyApplicant(option *ApplyOption) Applicant {
	return &godaddyApplicant{
		option: option,
	}
}

func (a *godaddyApplicant) Apply() (*Certificate, error) {
	access := &domain.GodaddyAccess{}
	json.Unmarshal([]byte(a.option.Access), access)

	config := godaddy.NewDefaultConfig()
	config.APIKey = access.ApiKey
	config.APISecret = access.ApiSecret
	if a.option.Timeout != 0 {
		config.PropagationTimeout = time.Duration(a.option.Timeout) * time.Second
	}

	provider, err := godaddy.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}

	return apply(a.option, provider)
}
