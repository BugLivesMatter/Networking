package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/google/uuid"

	"github.com/lab2/rest-api/internal/config"
)

// Sender отправляет письма через SMTP (ЛР8).
type Sender struct {
	cfg *config.Config
}

func NewSender(cfg *config.Config) *Sender {
	return &Sender{cfg: cfg}
}

// SendWelcome отправляет приветственное письмо (plain + HTML multipart/alternative).
func (s *Sender) SendWelcome(ctx context.Context, toEmail, displayName string, userID uuid.UUID) error {
	if displayName == "" {
		displayName = toEmail
	}
	subject := "Добро пожаловать в WP Labs"
	loginURL := s.cfg.AppPublicURL
	plain := fmt.Sprintf(
		"Здравствуйте, %s!\n\nРегистрация вашего аккаунта успешно завершена (идентификатор: %s).\n\nВойти в систему: %s\n",
		displayName,
		userID.String(),
		loginURL,
	)
	html := fmt.Sprintf(
		`<!DOCTYPE html><html><head><meta charset="UTF-8"></head><body>
<p>Здравствуйте, <strong>%s</strong>!</p>
<p>Регистрация вашего аккаунта успешно завершена.</p>
<p><a href="%s">Перейти ко входу в систему</a></p>
<p style="color:#666;font-size:12px;">Идентификатор пользователя: %s</p>
</body></html>`,
		escapeHTML(displayName),
		escapeHTML(loginURL),
		userID.String(),
	)
	msg, err := buildMultipartMail(s.cfg.SMTPFrom, toEmail, subject, plain, html)
	if err != nil {
		return err
	}
	host := s.cfg.SMTPHost
	addr := fmt.Sprintf("%s:%d", host, s.cfg.SMTPPort)
	var auth smtp.Auth
	if s.cfg.SMTPAuth == "login" {
		auth = newLoginAuth(s.cfg.SMTPUser, s.cfg.SMTPPass)
	} else {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, host)
	}

	if s.cfg.SMTPSecure && s.cfg.SMTPPort == 465 {
		return sendImplicitTLS(ctx, addr, host, auth, s.cfg.SMTPFrom, []string{toEmail}, msg)
	}
	return sendSTARTTLS(ctx, addr, host, auth, s.cfg.SMTPFrom, []string{toEmail}, msg)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func buildMultipartMail(from, to, subject, plainBody, htmlBody string) ([]byte, error) {
	boundary := "bWpLabsBoundary8"
	var buf bytes.Buffer
	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n",
		from, to, subject, boundary,
	)
	if _, err := buf.WriteString(headers); err != nil {
		return nil, err
	}
	fmt.Fprintf(&buf, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, plainBody)
	fmt.Fprintf(&buf, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", boundary, htmlBody)
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes(), nil
}

func sendImplicitTLS(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, msg []byte) error {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	tlsConn := tls.Client(conn, &tls.Config{ServerName: serverName})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp tls handshake: %w", err)
	}
	c, err := smtp.NewClient(tlsConn, serverName)
	if err != nil {
		_ = tlsConn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = c.Close() }()
	return sendMailClient(c, auth, from, to, msg)
}

func sendSTARTTLS(ctx context.Context, addr, serverName string, auth smtp.Auth, from string, to []string, msg []byte) error {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = c.Close() }()
	if ok, _ := c.Extension("STARTTLS"); ok {
		tcfg := &tls.Config{ServerName: serverName}
		if err := c.StartTLS(tcfg); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return sendMailClient(c, auth, from, to, msg)
}

func sendMailClient(c *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if ok, _ := c.Extension("AUTH"); ok {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", addr, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close writer: %w", err)
	}
	return c.Quit()
}

func isLocalSMTPHost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

// loginAuth реализует SMTP AUTH LOGIN; не сравнивает имя хоста с полем PlainAuth,
// но требует TLS (аналогично ограничению PlainAuth в стандартной библиотеке).
type loginAuth struct {
	username, password string
	step               int
}

func newLoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username: username, password: password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocalSMTPHost(server.Name) {
		return "", nil, errors.New("smtp LoginAuth: требуется TLS или localhost")
	}
	a.step = 0
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	a.step++
	switch a.step {
	case 1:
		return []byte(a.username), nil
	case 2:
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("smtp LoginAuth: неожиданный шаг %d", a.step)
	}
}
