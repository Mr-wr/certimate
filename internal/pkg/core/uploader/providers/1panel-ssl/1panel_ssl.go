package onepanelssl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/usual2970/certimate/internal/pkg/core/uploader"
	onepanelsdk "github.com/usual2970/certimate/internal/pkg/sdk3rd/1panel"
)

type UploaderConfig struct {
	// 1Panel 服务地址。
	ServerUrl string `json:"serverUrl"`
	// 1Panel 版本。
	ApiVersion string `json:"apiVersion"`
	// 1Panel 接口密钥。
	ApiKey string `json:"apiKey"`
	// 是否允许不安全的连接。
	AllowInsecureConnections bool `json:"allowInsecureConnections,omitempty"`
}

type UploaderProvider struct {
	config    *UploaderConfig
	logger    *slog.Logger
	sdkClient *onepanelsdk.Client
}

var _ uploader.Uploader = (*UploaderProvider)(nil)

func NewUploader(config *UploaderConfig) (*UploaderProvider, error) {
	if config == nil {
		panic("config is nil")
	}

	client, err := createSdkClient(config.ServerUrl, config.ApiVersion, config.ApiKey, config.AllowInsecureConnections)
	if err != nil {
		return nil, fmt.Errorf("failed to create sdk client: %w", err)
	}

	return &UploaderProvider{
		config:    config,
		logger:    slog.Default(),
		sdkClient: client,
	}, nil
}

func (u *UploaderProvider) WithLogger(logger *slog.Logger) uploader.Uploader {
	if logger == nil {
		u.logger = slog.New(slog.DiscardHandler)
	} else {
		u.logger = logger
	}
	return u
}

func (u *UploaderProvider) Upload(ctx context.Context, certPEM string, privkeyPEM string) (*uploader.UploadResult, error) {
	// 遍历证书列表，避免重复上传
	if res, err := u.findCertIfExists(ctx, certPEM, privkeyPEM); err != nil {
		return nil, err
	} else if res != nil {
		u.logger.Info("ssl certificate already exists")
		return res, nil
	}

	// 生成新证书名（需符合 1Panel 命名规则）
	certName := fmt.Sprintf("certimate-%d", time.Now().UnixMilli())

	// 上传证书
	uploadWebsiteSSLReq := &onepanelsdk.UploadWebsiteSSLRequest{
		Type:        "paste",
		Description: certName,
		Certificate: certPEM,
		PrivateKey:  privkeyPEM,
	}
	uploadWebsiteSSLResp, err := u.sdkClient.UploadWebsiteSSL(uploadWebsiteSSLReq)
	u.logger.Debug("sdk request '1panel.UploadWebsiteSSL'", slog.Any("request", uploadWebsiteSSLReq), slog.Any("response", uploadWebsiteSSLResp))
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request '1panel.UploadWebsiteSSL': %w", err)
	}

	// 遍历证书列表，获取刚刚上传证书 ID
	if res, err := u.findCertIfExists(ctx, certPEM, privkeyPEM); err != nil {
		return nil, err
	} else if res == nil {
		return nil, fmt.Errorf("no ssl certificate found, may be upload failed (code: %d, message: %s)", uploadWebsiteSSLResp.GetCode(), uploadWebsiteSSLResp.GetMessage())
	} else {
		return res, nil
	}
}

func (u *UploaderProvider) findCertIfExists(ctx context.Context, certPEM string, privkeyPEM string) (*uploader.UploadResult, error) {
	searchWebsiteSSLPageNumber := int32(1)
	searchWebsiteSSLPageSize := int32(100)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		searchWebsiteSSLReq := &onepanelsdk.SearchWebsiteSSLRequest{
			Page:     searchWebsiteSSLPageNumber,
			PageSize: searchWebsiteSSLPageSize,
		}
		searchWebsiteSSLResp, err := u.sdkClient.SearchWebsiteSSL(searchWebsiteSSLReq)
		u.logger.Debug("sdk request '1panel.SearchWebsiteSSL'", slog.Any("request", searchWebsiteSSLReq), slog.Any("response", searchWebsiteSSLResp))
		if err != nil {
			return nil, fmt.Errorf("failed to execute sdk request '1panel.SearchWebsiteSSL': %w", err)
		}

		for _, sslItem := range searchWebsiteSSLResp.Data.Items {
			if strings.TrimSpace(sslItem.PEM) == strings.TrimSpace(certPEM) &&
				strings.TrimSpace(sslItem.PrivateKey) == strings.TrimSpace(privkeyPEM) {
				// 如果已存在相同证书，直接返回
				return &uploader.UploadResult{
					CertId:   fmt.Sprintf("%d", sslItem.ID),
					CertName: sslItem.Description,
				}, nil
			}
		}

		if len(searchWebsiteSSLResp.Data.Items) < int(searchWebsiteSSLPageSize) {
			break
		} else {
			searchWebsiteSSLPageNumber++
		}
	}

	return nil, nil
}

func createSdkClient(serverUrl, apiVersion, apiKey string, skipTlsVerify bool) (*onepanelsdk.Client, error) {
	if _, err := url.Parse(serverUrl); err != nil {
		return nil, errors.New("invalid 1panel server url")
	}

	if apiVersion == "" {
		return nil, errors.New("invalid 1panel api version")
	}

	if apiKey == "" {
		return nil, errors.New("invalid 1panel api key")
	}

	client := onepanelsdk.NewClient(serverUrl, apiVersion, apiKey)
	if skipTlsVerify {
		client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}

	return client, nil
}
