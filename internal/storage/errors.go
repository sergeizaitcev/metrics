package storage

import "errors"

var (
	// ErrStorageClosed возвращается, если хранилище метрик было закрыто.
	ErrStorageClosed = errors.New("storage is closed")

	// ErrNotFound возвращается, когда метрика не найдена.
	ErrNotFound = errors.New("metric not found")
)
