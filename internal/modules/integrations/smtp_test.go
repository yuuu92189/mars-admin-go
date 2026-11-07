package integrations

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSMTPSend(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	messages := make(chan string, 1)
	go serveTestSMTP(t, listener, messages)

	address := listener.Addr().(*net.TCPAddr)
	provider, err := NewSMTP(SMTPConfig{
		Host: "127.0.0.1", Port: address.Port, From: "Mars <mars@example.invalid>", TLS: false,
		Dialer: &net.Dialer{Timeout: 2 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := provider.Send(ctx, "user@example.invalid", "Test subject", "Test body"); err != nil {
		t.Fatal(err)
	}
	message := <-messages
	if !strings.Contains(message, "Subject: Test subject") || !strings.Contains(message, "Test body") {
		t.Fatalf("unexpected SMTP message %q", message)
	}
}

func serveTestSMTP(t *testing.T, listener net.Listener, messages chan<- string) {
	t.Helper()
	connection, err := listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = connection.Close() }()
	reader := bufio.NewReader(connection)
	writer := bufio.NewWriter(connection)
	writeLine := func(value string) {
		_, _ = writer.WriteString(value + "\r\n")
		_ = writer.Flush()
	}
	writeLine("220 smtp.test ESMTP")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			writeLine("250-smtp.test")
			writeLine("250 OK")
		case strings.HasPrefix(command, "MAIL FROM"), strings.HasPrefix(command, "RCPT TO"):
			writeLine("250 OK")
		case command == "DATA":
			writeLine("354 End data with <CR><LF>.<CR><LF>")
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			messages <- data.String()
			writeLine("250 Queued")
		case command == "QUIT":
			writeLine("221 Bye")
			return
		default:
			writeLine("250 OK")
		}
	}
}
