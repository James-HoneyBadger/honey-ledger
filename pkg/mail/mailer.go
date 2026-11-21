// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package mail

// Mailer is email sender interface
type Mailer interface {
	SendMail(message *MailMessage) error
}
