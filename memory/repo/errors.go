package repo

import "errors"

// ErrExists is returned when a create operation would duplicate a record.
var ErrExists = errors.New("memory: record already exists")
