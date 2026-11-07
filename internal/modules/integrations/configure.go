package integrations

import (
	"fmt"
	"log/slog"

	"mars-admin-go/internal/platform/config"
)

func Configure(cfg config.IntegrationConfig, logger *slog.Logger, production bool) (*Registry, SMSProvider, error) {
	registry := NewRegistry()
	var sms SMSProvider
	switch cfg.SMSProvider {
	case "", "console":
		if production {
			return nil, nil, fmt.Errorf("console SMS is not allowed in production")
		}
		console := NewConsoleSMS(logger)
		registry.RegisterSMS("console", console)
		sms = console
	case "aliyun":
		provider, err := NewAliyunSMS(AliyunSMSConfig{
			Endpoint: cfg.AliyunSMS.Endpoint, AccessKeyID: cfg.AliyunSMS.AccessKeyID,
			AccessKeySecret: cfg.AliyunSMS.AccessKeySecret, SignName: cfg.AliyunSMS.SignName,
			TemplateCode: cfg.AliyunSMS.TemplateCode, NoticeTemplate: cfg.AliyunSMS.NoticeTemplate,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("configure Aliyun SMS: %w", err)
		}
		registry.RegisterSMS("aliyun", provider)
		sms = provider
	default:
		return nil, nil, fmt.Errorf("unsupported SMS provider %q", cfg.SMSProvider)
	}
	if cfg.WeChatMini.AppID != "" && cfg.WeChatMini.AppSecret != "" {
		provider, err := NewWeChatMiniProgram(WeChatSocialConfig{
			AppID: cfg.WeChatMini.AppID, AppSecret: cfg.WeChatMini.AppSecret,
			Endpoint: cfg.WeChatMini.Endpoint,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("configure WeChat mini program: %w", err)
		}
		registry.RegisterSocial("wechat_miniprogram", provider)
	}
	if cfg.PushWebhook.URL != "" {
		provider, err := NewWebhookPush(WebhookPushConfig{
			Provider: cfg.PushWebhook.Provider, URL: cfg.PushWebhook.URL, Secret: cfg.PushWebhook.Secret,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("configure push webhook: %w", err)
		}
		registry.RegisterPush(cfg.PushWebhook.Provider, provider)
	}
	if cfg.WeChatPay.MerchantID != "" {
		provider, err := NewWeChatPay(WeChatPayConfig{
			Endpoint: cfg.WeChatPay.Endpoint, MerchantID: cfg.WeChatPay.MerchantID,
			AppID: cfg.WeChatPay.AppID, SerialNumber: cfg.WeChatPay.SerialNumber,
			PrivateKeyPEM:        cfg.WeChatPay.PrivateKeyPEM,
			PlatformPublicKeyPEM: cfg.WeChatPay.PlatformPublicKeyPEM,
			PlatformSerialNumber: cfg.WeChatPay.PlatformSerialNumber,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("configure WeChat Pay: %w", err)
		}
		registry.RegisterPayment("wechat", provider)
	}
	if cfg.Alipay.AppID != "" {
		provider, err := NewAlipay(AlipayConfig{
			Endpoint: cfg.Alipay.Endpoint, AppID: cfg.Alipay.AppID,
			PrivateKeyPEM: cfg.Alipay.PrivateKeyPEM, PublicKeyPEM: cfg.Alipay.PublicKeyPEM,
			NotifyURL: cfg.Alipay.NotifyURL, ReturnURL: cfg.Alipay.ReturnURL,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("configure Alipay: %w", err)
		}
		registry.RegisterPayment("alipay", provider)
	}
	if cfg.SMTP.Host != "" || cfg.SMTP.From != "" {
		provider, err := NewSMTP(SMTPConfig{
			Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username,
			Password: cfg.SMTP.Password, From: cfg.SMTP.From, TLS: cfg.SMTP.TLS,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("configure SMTP: %w", err)
		}
		registry.RegisterMail("smtp", provider)
	}
	return registry, sms, nil
}
