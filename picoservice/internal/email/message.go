package email

import (
	"bytes"
	"fmt"
	"html"
	"time"
)

const verificationBoundary = "picoservice-verification"

// BuildVerificationMessage returns a multipart verification email.
func BuildVerificationMessage(from, to, name, verifyURL string) []byte {
	var message bytes.Buffer

	fmt.Fprintf(&message, "From: %s\r\n", from)
	fmt.Fprintf(&message, "To: %s\r\n", to)
	message.WriteString("Subject: Verify your email\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&message, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n", verificationBoundary)
	message.WriteString("\r\n")

	fmt.Fprintf(&message, "--%s\r\n", verificationBoundary)
	message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	message.WriteString("\r\n")
	fmt.Fprintf(&message, "Hi %s,\r\n\r\n", name)
	message.WriteString("Verify your email by opening this link:\r\n\r\n")
	fmt.Fprintf(&message, "%s\r\n\r\n", verifyURL)
	message.WriteString("This link expires in 24 hours. If you did not create an account, ignore this email.\r\n")
	message.WriteString("\r\n")

	fmt.Fprintf(&message, "--%s\r\n", verificationBoundary)
	message.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	message.WriteString("\r\n")
	fmt.Fprintf(&message, "<p>Hi %s,</p>\r\n", html.EscapeString(name))
	fmt.Fprintf(&message, "<p><a href=%q>Verify your email</a></p>\r\n", html.EscapeString(verifyURL))
	message.WriteString("<p>This link expires in 24 hours. If you did not create an account, ignore this email.</p>\r\n")
	fmt.Fprintf(&message, "\r\n--%s--\r\n", verificationBoundary)

	return message.Bytes()
}
