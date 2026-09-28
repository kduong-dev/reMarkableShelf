package database

// RowScanner is a *sql.Row or *sql.Rows, so one scan function can read a
// single row and a row of many.
type RowScanner interface {
	Scan(dest ...any) error
}
