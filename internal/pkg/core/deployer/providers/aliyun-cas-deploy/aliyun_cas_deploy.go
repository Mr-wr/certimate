﻿package aliyuncasdeploy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	aliyunCas "github.com/alibabacloud-go/cas-20200407/v3/client"
	aliyunOpen "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	xerrors "github.com/pkg/errors"

	"github.com/usual2970/certimate/internal/pkg/core/deployer"
	"github.com/usual2970/certimate/internal/pkg/core/logger"
	"github.com/usual2970/certimate/internal/pkg/core/uploader"
	uploadersp "github.com/usual2970/certimate/internal/pkg/core/uploader/providers/aliyun-cas"
)

type AliyunCASDeployDeployerConfig struct {
	// 阿里云 AccessKeyId。
	AccessKeyId string `json:"accessKeyId"`
	// 阿里云 AccessKeySecret。
	AccessKeySecret string `json:"accessKeySecret"`
	// 阿里云地域。
	Region string `json:"region"`
	// 阿里云云产品资源 ID 数组。
	ResourceIds []string `json:"resourceIds"`
	// 阿里云云联系人 ID 数组。
	// 零值时默认使用账号下第一个联系人。
	ContactIds []string `json:"contactIds"`
}

type AliyunCASDeployDeployer struct {
	config      *AliyunCASDeployDeployerConfig
	logger      logger.Logger
	sdkClient   *aliyunCas.Client
	sslUploader uploader.Uploader
}

var _ deployer.Deployer = (*AliyunCASDeployDeployer)(nil)

func New(config *AliyunCASDeployDeployerConfig) (*AliyunCASDeployDeployer, error) {
	return NewWithLogger(config, logger.NewNilLogger())
}

func NewWithLogger(config *AliyunCASDeployDeployerConfig, logger logger.Logger) (*AliyunCASDeployDeployer, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	if logger == nil {
		return nil, errors.New("logger is nil")
	}

	client, err := createSdkClient(config.AccessKeyId, config.AccessKeySecret, config.Region)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to create sdk client")
	}

	uploader, err := createSslUploader(config.AccessKeyId, config.AccessKeySecret, config.Region)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to create ssl uploader")
	}

	return &AliyunCASDeployDeployer{
		logger:      logger,
		config:      config,
		sdkClient:   client,
		sslUploader: uploader,
	}, nil
}

func (d *AliyunCASDeployDeployer) Deploy(ctx context.Context, certPem string, privkeyPem string) (*deployer.DeployResult, error) {
	if len(d.config.ResourceIds) == 0 {
		return nil, errors.New("config `resourceIds` is required")
	}

	// 上传证书到 CAS
	upres, err := d.sslUploader.Upload(ctx, certPem, privkeyPem)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to upload certificate file")
	}

	d.logger.Logt("certificate file uploaded", upres)

	contactIds := d.config.ContactIds
	if len(contactIds) == 0 {
		// 获取联系人列表
		// REF: https://help.aliyun.com/zh/ssl-certificate/developer-reference/api-cas-2020-04-07-listcontact
		listContactReq := &aliyunCas.ListContactRequest{}
		listContactReq.ShowSize = tea.Int32(1)
		listContactReq.CurrentPage = tea.Int32(1)
		listContactResp, err := d.sdkClient.ListContact(listContactReq)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to execute sdk request 'cas.ListContact'")
		}

		if len(listContactResp.Body.ContactList) > 0 {
			contactIds = []string{fmt.Sprintf("%d", listContactResp.Body.ContactList[0].ContactId)}
		}
	}

	// 创建部署任务
	// REF: https://help.aliyun.com/zh/ssl-certificate/developer-reference/api-cas-2020-04-07-createdeploymentjob
	createDeploymentJobReq := &aliyunCas.CreateDeploymentJobRequest{
		Name:        tea.String(fmt.Sprintf("certimate-%d", time.Now().UnixMilli())),
		JobType:     tea.String("user"),
		CertIds:     tea.String(upres.CertId),
		ResourceIds: tea.String(strings.Join(d.config.ResourceIds, ",")),
		ContactIds:  tea.String(strings.Join(contactIds, ",")),
	}
	createDeploymentJobResp, err := d.sdkClient.CreateDeploymentJob(createDeploymentJobReq)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to execute sdk request 'cas.CreateDeploymentJob'")
	}

	d.logger.Logt("已创建部署任务", createDeploymentJobResp)

	// 循环获取部署任务详情，等待任务状态变更
	// REF: https://help.aliyun.com/zh/ssl-certificate/developer-reference/api-cas-2020-04-07-describedeploymentjob
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		describeDeploymentJobReq := &aliyunCas.DescribeDeploymentJobRequest{
			JobId: createDeploymentJobResp.Body.JobId,
		}
		describeDeploymentJobResp, err := d.sdkClient.DescribeDeploymentJob(describeDeploymentJobReq)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to execute sdk request 'cas.DescribeDeploymentJob'")
		}

		if describeDeploymentJobResp.Body.Status == nil || *describeDeploymentJobResp.Body.Status == "editing" {
			return nil, errors.New("部署任务状态异常")
		}

		if *describeDeploymentJobResp.Body.Status == "success" || *describeDeploymentJobResp.Body.Status == "error" {
			d.logger.Logt("已获取部署任务详情", describeDeploymentJobResp)
			break
		}

		d.logger.Logt("部署任务未完成 ...")
		time.Sleep(time.Second * 5)
	}

	return &deployer.DeployResult{}, nil
}

func createSdkClient(accessKeyId, accessKeySecret, region string) (*aliyunCas.Client, error) {
	if region == "" {
		region = "cn-hangzhou" // CAS 服务默认区域：华东一杭州
	}

	// 接入点一览 https://help.aliyun.com/zh/ssl-certificate/developer-reference/endpoints
	var endpoint string
	switch region {
	case "cn-hangzhou":
		endpoint = "cas.aliyuncs.com"
	default:
		endpoint = fmt.Sprintf("cas.%s.aliyuncs.com", region)
	}

	config := &aliyunOpen.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	client, err := aliyunCas.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func createSslUploader(accessKeyId, accessKeySecret, region string) (uploader.Uploader, error) {
	uploader, err := uploadersp.New(&uploadersp.AliyunCASUploaderConfig{
		AccessKeyId:     accessKeyId,
		AccessKeySecret: accessKeySecret,
		Region:          region,
	})
	return uploader, err
}
