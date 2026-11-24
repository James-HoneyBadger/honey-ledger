// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"bytes"
	"errors"
	"io"

	"github.com/abemedia/go-cabinet"
)

const (
	// microsoftMoneyBackupSignature is the Microsoft Cabinet signature ("MSCF")
	// that every Microsoft Money backup file starts with.
	microsoftMoneyBackupSignature = "MSCF"

	// microsoftMoneyBackupProbeSize is how many leading bytes of a cabinet member
	// are read to recognise the Microsoft Money database inside it.
	microsoftMoneyBackupProbeSize = 64
)

var (
	// maxMicrosoftMoneyBackupDatabaseSize bounds how large the database inside a
	// backup may be. A real Money database is a few tens of megabytes, so this is
	// roomy while still refusing an archive that expands into a decompression
	// bomb.
	maxMicrosoftMoneyBackupDatabaseSize int64 = 512 << 20

	// maxMicrosoftMoneyBackupFiles bounds how many cabinet members are examined,
	// so a crafted archive cannot make the importer work through thousands of
	// entries.
	maxMicrosoftMoneyBackupFiles = 32
)

// errMicrosoftMoneyBackupMemberIsNotDatabase reports a cabinet member that does
// not carry a Microsoft Money database header, which is how every member other
// than the database is treated.
var errMicrosoftMoneyBackupMemberIsNotDatabase = errors.New("the cabinet member is not a microsoft money database")

// IsMicrosoftMoneyBackupFile returns whether the data is a Microsoft Cabinet
// archive. Microsoft Money writes a backup (`.mbf`) file as a cabinet that
// wraps the database.
func IsMicrosoftMoneyBackupFile(data []byte) bool {
	return len(data) >= len(microsoftMoneyBackupSignature) && string(data[:len(microsoftMoneyBackupSignature)]) == microsoftMoneyBackupSignature
}

// extractMicrosoftMoneyDatabaseFromBackup returns the database held in a
// Microsoft Money backup archive. The archive is expanded in memory, so nothing
// is written to disk and nothing has to be cleaned up afterwards.
func extractMicrosoftMoneyDatabaseFromBackup(ctx Context, data []byte) ([]byte, error) {
	archive, err := cabinet.NewReader(bytes.NewReader(data))

	if err != nil {
		errorf(ctx, "[microsoft_money_backup_archive.extractMicrosoftMoneyDatabaseFromBackup] cannot read the Microsoft Money backup archive, because %s", err.Error())
		return nil, ErrBackupFileNotReadable
	}

	if len(archive.Files) < 1 || len(archive.Files) > maxMicrosoftMoneyBackupFiles {
		errorf(ctx, "[microsoft_money_backup_archive.extractMicrosoftMoneyDatabaseFromBackup] the backup archive holds %d files, which is outside the supported range", len(archive.Files))
		return nil, ErrBackupFileNotReadable
	}

	// Money does not name the member after the file it holds, so the database is
	// recognised by its header rather than by its name.
	for _, file := range archive.Files {
		database, err := readMicrosoftMoneyDatabaseFromBackupMember(file)

		if errors.Is(err, errMicrosoftMoneyBackupMemberIsNotDatabase) {
			continue
		}

		if err != nil {
			return nil, err
		}

		infof(ctx, "[microsoft_money_backup_archive.extractMicrosoftMoneyDatabaseFromBackup] extracted %d bytes of the Microsoft Money database from the backup archive", len(database))
		return database, nil
	}

	errorf(ctx, "[microsoft_money_backup_archive.extractMicrosoftMoneyDatabaseFromBackup] the backup archive holds no Microsoft Money database")
	return nil, ErrBackupFileNotReadable
}

// readMicrosoftMoneyDatabaseFromBackupMember returns the database held by one
// cabinet member, or errMicrosoftMoneyBackupMemberIsNotDatabase when the member
// holds something else.
func readMicrosoftMoneyDatabaseFromBackupMember(file *cabinet.File) ([]byte, error) {
	// The declared size is checked before decompressing anything, so an archive
	// that claims to expand into gigabytes is refused without the work.
	if int64(file.Size()) > maxMicrosoftMoneyBackupDatabaseSize {
		return nil, ErrBackupFileTooLarge
	}

	reader, err := file.Open()

	if err != nil {
		return nil, ErrBackupFileNotReadable
	}

	defer reader.Close()

	probe := make([]byte, microsoftMoneyBackupProbeSize)
	probeLength, err := io.ReadFull(reader, probe)

	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, ErrBackupFileNotReadable
	}

	if !IsMicrosoftMoneyDataFile(probe[:probeLength]) {
		return nil, errMicrosoftMoneyBackupMemberIsNotDatabase
	}

	// Read one byte past the limit, so a member that expands beyond it is
	// detected rather than silently truncated.
	remaining, err := io.ReadAll(io.LimitReader(reader, maxMicrosoftMoneyBackupDatabaseSize-int64(probeLength)+1))

	if err != nil {
		return nil, ErrBackupFileNotReadable
	}

	if int64(probeLength)+int64(len(remaining)) > maxMicrosoftMoneyBackupDatabaseSize {
		return nil, ErrBackupFileTooLarge
	}

	database := make([]byte, 0, probeLength+len(remaining))
	database = append(database, probe[:probeLength]...)
	database = append(database, remaining...)

	return database, nil
}
