package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v3"
)

func TestSaveObject(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO storage_objects`).
		WithArgs(pgxmock.AnyArg(), "user-1", "https://storage.example/file", "photo").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewService(mock)
	id, err := svc.SaveObject(context.Background(), "user-1", "https://storage.example/file", "photo")
	if err != nil {
		t.Fatalf("save object: %v", err)
	}
	if id == "" {
		t.Fatalf("expected id")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaveObjectError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO storage_objects`).
		WithArgs(pgxmock.AnyArg(), "user-1", "url", "kind").
		WillReturnError(errSave)

	svc := NewService(mock)
	_, err = svc.SaveObject(context.Background(), "user-1", "url", "kind")
	if err == nil {
		t.Fatalf("expected error")
	}
}

var errSave = errors.New("save error")
