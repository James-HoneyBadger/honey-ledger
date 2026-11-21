// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package mail

// MailMessage represents an email entity
type MailMessage struct {
	To      string
	Subject string
	Body    string
}
