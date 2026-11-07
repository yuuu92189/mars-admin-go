package integrations

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	From        string
	TLS         bool
	ImplicitTLS bool
	Dialer      *net.Dialer
}

type SMTP struct {
	config SMTPConfig
	dialer *net.Dialer
}

func NewSMTP(config SMTPConfig) (*SMTP, error) {
	if config.Host == "" || config.From == "" {
		return nil, ErrNotConfigured
	}
	if _, err := mail.ParseAddress(config.From); err != nil {
		return nil, fmt.Errorf("invalid SMTP from address: %w", err)
	}
	if config.Port <= 0 {
		config.Port = 587
	}
	if config.Dialer == nil {
		config.Dialer = &net.Dialer{Timeout: 10 * time.Second}
	}
	return &SMTP{config: config, dialer: config.Dialer}, nil
}

func (s *SMTP) Send(ctx context.Context, to, subject, body string) error {
	recipient, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	subject = sanitizeHeader(subject)
	from, _ := mail.ParseAddress(s.config.From)
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	connection, err := s.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial SMTP: %w", err)
	}
	defer func() { _ = connection.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	if s.config.ImplicitTLS {
		tlsConnection := tls.Client(connection, &tls.Config{
			ServerName: s.config.Host,
			MinVersion: tls.VersionTLS12,
		})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("SMTP TLS handshake: %w", err)
		}
		connection = tlsConnection
	}
	client, err := smtp.NewClient(connection, s.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer func() { _ = client.Close() }()
	if s.config.TLS && !s.config.ImplicitTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if s.config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)); err != nil {
			return fmt.Errorf("SMTP authentication: %w", err)
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := "From: " + s.config.From + "\r\n" +
		"To: " + recipient.Address + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + body + "\r\n"
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}

var _ MailProvider = (*SMTP)(nil)
