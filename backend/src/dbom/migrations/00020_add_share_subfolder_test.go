package migrations

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

const subfolderColumnQuery = `SELECT COUNT\(\*\) FROM pragma_table_info\('exported_shares'\) WHERE name = 'subfolder'`

func expectSubfolderColumn(mock sqlmock.Sqlmock, count int) {
	mock.ExpectQuery(subfolderColumnQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

func TestUp00020AddsSubfolderColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSubfolderColumn(mock, 0)
	mock.ExpectExec(`ALTER TABLE exported_shares ADD COLUMN subfolder TEXT NOT NULL DEFAULT ''`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, Up00020(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUp00020SkipsExistingColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSubfolderColumn(mock, 1)

	require.NoError(t, Up00020(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUp00020ReturnsLookupError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(subfolderColumnQuery).WillReturnError(errors.New("db locked"))

	require.ErrorContains(t, Up00020(context.Background(), db), "db locked")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDown00020DropsSubfolderColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSubfolderColumn(mock, 1)
	mock.ExpectExec(`ALTER TABLE exported_shares DROP COLUMN subfolder`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, Down00020(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDown00020SkipsMissingColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSubfolderColumn(mock, 0)

	require.NoError(t, Down00020(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}
