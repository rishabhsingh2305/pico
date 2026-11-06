package email

import (
	"bytes"
	"fmt"
	"html"
	"time"
)

const verificationBoundary = "picoservice-verification"

const verificationHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Verify your email</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f1ea;">
<div style="display:none;font-size:1px;line-height:1px;max-height:0;max-width:0;opacity:0;overflow:hidden;color:#f4f1ea;">Confirm your email address for Pico.</div>
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f4f1ea;">
  <tr>
    <td align="center" style="padding:32px 16px;">
      <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="max-width:560px;background-color:#fbf8f1;border:1px solid #d8d2c4;border-radius:8px;">
        <tr>
          <td style="padding:32px 32px 0 32px;">
            <table role="presentation" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td width="36" height="36" align="center" valign="middle" style="width:36px;height:36px;background-color:#c63c1e;border-radius:6px;color:#fff8f0;font-family:Georgia,'Times New Roman',serif;font-size:18px;font-weight:700;">P</td>
                <td style="padding-left:10px;font-family:'Bricolage Grotesque',Georgia,'Times New Roman',serif;font-size:24px;font-weight:700;letter-spacing:-0.04em;color:#11100e;">Pico</td>
              </tr>
            </table>
          </td>
        </tr>
        <tr>
          <td style="padding:28px 32px 0 32px;">
            <p style="margin:0 0 8px 0;font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:11px;font-weight:600;letter-spacing:0.18em;text-transform:uppercase;color:#6f6a60;">Expense workspace</p>
            <h1 style="margin:0;font-family:'Bricolage Grotesque',Georgia,'Times New Roman',serif;font-size:30px;line-height:1.1;letter-spacing:-0.03em;font-weight:700;color:#11100e;">Verify your email</h1>
          </td>
        </tr>
        <tr>
          <td style="padding:16px 32px 0 32px;font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:16px;line-height:1.6;color:#2a2620;">
            <p style="margin:0 0 12px 0;">Hi %s,</p>
            <p style="margin:0 0 24px 0;">Confirm your email address to finish setting up your Pico account.</p>
          </td>
        </tr>
        <tr>
          <td style="padding:0 32px;">
            <table role="presentation" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td align="center" style="background-color:#c63c1e;border-radius:6px;">
                  <a href="%s" target="_blank" style="display:inline-block;padding:13px 26px;font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:15px;font-weight:600;line-height:1;color:#fff8f0;text-decoration:none;">Verify email</a>
                </td>
              </tr>
            </table>
          </td>
        </tr>
        <tr>
          <td style="padding:20px 32px 0 32px;font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:13px;line-height:1.6;color:#6f6a60;">
            <p style="margin:0 0 8px 0;">If the button does not work, paste this link into your browser:</p>
            <p style="margin:0;word-break:break-all;"><a href="%s" target="_blank" style="color:#c63c1e;text-decoration:underline;">%s</a></p>
          </td>
        </tr>
        <tr>
          <td style="padding:24px 32px 32px 32px;">
            <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="border-top:1px solid #d8d2c4;">
              <tr>
                <td style="padding-top:16px;font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:13px;line-height:1.6;color:#6f6a60;">
                  <p style="margin:0 0 6px 0;">This link expires in 24 hours. If you did not create an account, you can ignore this email.</p>
                  <p style="margin:0;font-size:12px;color:#6f6a60;">Pico &middot; Shared expense tracking for small teams</p>
                </td>
              </tr>
            </table>
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>
</body>
</html>
`

// BuildVerificationMessage returns a multipart verification email.
func BuildVerificationMessage(from, to, name, verifyURL string) []byte {
	var message bytes.Buffer

	fmt.Fprintf(&message, "From: %s\r\n", from)
	fmt.Fprintf(&message, "To: %s\r\n", to)
	message.WriteString("Subject: Verify your email - Pico\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&message, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n", verificationBoundary)
	message.WriteString("\r\n")

	fmt.Fprintf(&message, "--%s\r\n", verificationBoundary)
	message.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	message.WriteString("\r\n")
	fmt.Fprintf(&message, "Hi %s,\r\n\r\n", name)
	message.WriteString("Confirm your email address to finish setting up your Pico account:\r\n\r\n")
	fmt.Fprintf(&message, "%s\r\n\r\n", verifyURL)
	message.WriteString("This link expires in 24 hours. If you did not create an account, ignore this email.\r\n")
	message.WriteString("\r\n")

	fmt.Fprintf(&message, "--%s\r\n", verificationBoundary)
	message.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	message.WriteString("\r\n")

	escapedName := html.EscapeString(name)
	escapedURL := html.EscapeString(verifyURL)
	fmt.Fprintf(&message, verificationHTML, escapedName, escapedURL, escapedURL, escapedURL)
	fmt.Fprintf(&message, "\r\n--%s--\r\n", verificationBoundary)

	return message.Bytes()
}
