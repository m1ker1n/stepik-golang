package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/m1ker1n/go-generics"
	"net/http"
	"strings"
)

type HttpResponse struct {
	Error    error `json:"error,omitempty"`
	Response any   `json:"response,omitempty"`
}

func (r HttpResponse) Write(w http.ResponseWriter, status int) {
	result, _ := json.Marshal(r)
	w.WriteHeader(status)
	w.Write(result)
}

type Column struct {
	Field      string
	Type       string
	Collation  sql.Null[string]
	Null       string
	Key        string
	Default    sql.Null[string]
	Extra      string
	Privileges string
	Comment    string
}

type MySQLDBExplorerService struct {
	db *sql.DB

	tablesColumns map[string][]Column
}

func NewMySQLDBExplorerService(db *sql.DB) (*MySQLDBExplorerService, error) {
	tables, err := getTables(db)
	if err != nil {
		return nil, err
	}

	tableColumns := make(map[string][]Column, len(tables))
	for _, table := range tables {
		columns, err := getTableColumns(db, table)
		if err != nil {
			return nil, err
		}
		tableColumns[table] = columns
	}

	return &MySQLDBExplorerService{
		db:            db,
		tablesColumns: tableColumns,
	}, nil
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

func getTableColumns(db *sql.DB, table string) ([]Column, error) {
	// here must be SQL-injection but well I only do it for initialising service
	// also this function is private, what would go wrong? :)
	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column

	for rows.Next() {
		var c Column
		if err := rows.Scan(
			&c.Field,
			&c.Type,
			&c.Collation,
			&c.Null,
			&c.Key,
			&c.Default,
			&c.Extra,
			&c.Privileges,
			&c.Comment,
		); err != nil {
			return nil, err
		}

		columns = append(columns, c)
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

func (s *MySQLDBExplorerService) GetTableRows(ctx context.Context, table string, limit, offset int) ([]map[string]any, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return nil, fmt.Errorf("table %s doesn't exists", table)
	}

	results := make([]map[string]any, 0, limit)
	// At the moment we checked if table is valid with looking up in s.tablesColumns[table].
	// So there can't be SQL-injection in the table variable.
	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT ?, ?", table)
	rows, err := s.db.QueryContext(ctx, query, offset, limit)
	if err != nil {
		return nil, err
	}

	//fieldsColumnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		scanInto := make([]any, len(columns))
		//I must initialize with new(any), without this it doesn't work
		for i := range scanInto {
			scanInto[i] = new(any)
		}

		if err := rows.Scan(scanInto...); err != nil {
			return nil, err
		}

		for i := range scanInto {
			valPointer, ok := (scanInto[i]).(*any)
			if !ok {
				return nil, errors.New("couldn't type assert wtf")
			}
			val := *valPointer

			// Here might be a problem because in db can be strings values with UTF-8
			if strVal, ok := (val).([]uint8); ok {
				scanInto[i] = string(strVal)
			} else {
				scanInto[i] = val
			}
		}

		rowResult := make(map[string]any)
		for i, column := range columns {
			rowResult[column.Field] = scanInto[i]
		}
		results = append(results, rowResult)
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *MySQLDBExplorerService) GetTableRow(ctx context.Context, table string, key any) (map[string]any, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return nil, fmt.Errorf("table %s doesn't exists", table)
	}

	primaryKeyColumn, ok := generics.FindFirst(columns, func(x Column) bool {
		return x.Key == "PRI"
	})
	if !ok {
		return nil, errors.New("couldn't find primary key field")
	}
	// At the moment we checked if table is valid with looking up in s.tablesColumns[table].
	// So there can't be SQL-injection in the table variable.
	// Analogically with primaryKeyColumn.Field.
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE %s = ?", table, primaryKeyColumn.Field)
	row := s.db.QueryRowContext(ctx, query, key)

	scanInto := make([]any, len(columns))
	//I must initialize with new(any), without this it doesn't work
	for i := range scanInto {
		scanInto[i] = new(any)
	}

	if err := row.Scan(scanInto...); err != nil {
		return nil, err
	}

	for i := range scanInto {
		valPointer, ok := (scanInto[i]).(*any)
		if !ok {
			return nil, errors.New("couldn't type assert wtf")
		}
		val := *valPointer

		// Here might be a problem because in db can be strings values with UTF-8
		if strVal, ok := (val).([]uint8); ok {
			scanInto[i] = string(strVal)
		} else {
			scanInto[i] = val
		}
	}

	result := make(map[string]any)
	for i, column := range columns {
		result[column.Field] = scanInto[i]
	}

	return result, nil
}

func (s *MySQLDBExplorerService) CreateTableRow(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return nil, fmt.Errorf("table %s doesn't exists", table)
	}

	dataFields := generics.MapKeys(data)
	allowedFields := generics.Filter(dataFields, func(x string) bool {
		for _, column := range columns {
			if column.Field == x && column.Extra != "auto_increment" {
				return true
			}
		}
		return false
	})

	colNames := fmt.Sprintf("`%s`", strings.Join(allowedFields, "`, `"))
	placeholders := strings.Join(strings.Split(strings.Repeat("?", len(allowedFields)), ""), ", ")
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUE (%s)", table, colNames, placeholders)

	args := make([]any, len(allowedFields))
	for i, allowedField := range allowedFields {
		args[i] = data[allowedField]
	}
	sqlResult, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	insertedId, err := sqlResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	result := make(map[string]any)
	result["updated"] = insertedId

	return result, nil
}

func (s *MySQLDBExplorerService) UpdateTableRow(ctx context.Context, table string, key any, data map[string]any) (map[string]any, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return nil, fmt.Errorf("table %s doesn't exists", table)
	}

	primaryKeyColumn, ok := generics.FindFirst(columns, func(x Column) bool {
		return x.Key == "PRI"
	})
	if !ok {
		return nil, errors.New("primary key column is not found")
	}
	dataFields := generics.MapKeys(data)
	allowedFields := generics.Filter(dataFields, func(x string) bool {
		for _, column := range columns {
			if column.Field == x && column.Extra != "auto_increment" {
				return true
			}
		}
		return false
	})

	assignments := generics.Map(allowedFields, func(field string) (string, error) {
		return fmt.Sprintf("`%s`=?", field), nil
	})
	assignmentList := strings.Join(assignments, ", ")
	whereCondition := fmt.Sprintf("`%s`=?", primaryKeyColumn.Field)
	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE %s", table, assignmentList, whereCondition)

	args := make([]any, len(allowedFields)+1)
	for i, allowedField := range allowedFields {
		args[i] = data[allowedField]
	}
	args[len(allowedFields)] = key
	sqlResult, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	affectedRows, err := sqlResult.RowsAffected()
	if err != nil {
		return nil, err
	}

	result := make(map[string]any)
	result["updated"] = affectedRows

	return result, nil
}

func (s *MySQLDBExplorerService) DeleteTableRow(ctx context.Context, table string, key any) (map[string]any, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return nil, fmt.Errorf("table %s doesn't exists", table)
	}

	primaryKeyColumn, ok := generics.FindFirst(columns, func(x Column) bool {
		return x.Key == "PRI"
	})
	if !ok {
		return nil, errors.New("couldn't find primary key field")
	}
	// At the moment we checked if table is valid with looking up in s.tablesColumns[table].
	// So there can't be SQL-injection in the table variable.
	// Analogically with primaryKeyColumn.Field.
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, primaryKeyColumn.Field)
	sqlResult, err := s.db.ExecContext(ctx, query, key)
	if err != nil {
		return nil, err
	}

	rowsDeleted, err := sqlResult.RowsAffected()
	if err != nil {
		return nil, err
	}

	result := make(map[string]any)
	result["deleted"] = rowsDeleted

	return result, nil
}

type DBExplorerService interface {
	GetTableRows(ctx context.Context, table string, limit, offset int) ([]map[string]any, error)
	GetTableRow(ctx context.Context, table string, key any) (map[string]any, error)
	CreateTableRow(ctx context.Context, table string, data map[string]any) (map[string]any, error)
	UpdateTableRow(ctx context.Context, table string, key any, data map[string]any) (map[string]any, error)
	DeleteTableRow(ctx context.Context, table string, key any) (map[string]any, error)
}

type DBExplorer struct {
	service DBExplorerService
}

func NewDbExplorer(db *sql.DB) (*DBExplorer, error) {
	service, err := NewMySQLDBExplorerService(db)
	if err != nil {
		return nil, err
	}

	return &DBExplorer{
		service: service,
	}, nil
}

var (
	ErrMethodNotFound = errors.New("method for this url is not found")
)

func (srv *DBExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		switch r.Method {
		case http.MethodGet:
			srv.GetAllTables(w, r)
		default:
			HttpResponse{Error: ErrMethodNotFound}.Write(w, http.StatusMethodNotAllowed)
		}
	case "/{table}":
		switch r.Method {
		case http.MethodGet:
			srv.GetTableRows(w, r)
		case http.MethodPost:
			srv.PostTableRow(w, r)
		default:
			HttpResponse{Error: ErrMethodNotFound}.Write(w, http.StatusMethodNotAllowed)
		}
	case "/{table}/{id}":
		switch r.Method {
		case http.MethodGet:
			srv.GetTableRow(w, r)
		case http.MethodPut:
			srv.PutTableRow(w, r)
		case http.MethodDelete:
			srv.DeleteTableRow(w, r)
		default:
			HttpResponse{Error: ErrMethodNotFound}.Write(w, http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

func (srv *DBExplorer) GetAllTables(w http.ResponseWriter, r *http.Request) {

}

func (srv *DBExplorer) GetTableRows(w http.ResponseWriter, r *http.Request) {

}

func (srv *DBExplorer) GetTableRow(w http.ResponseWriter, r *http.Request) {

}

func (srv *DBExplorer) PostTableRow(w http.ResponseWriter, r *http.Request) {

}

func (srv *DBExplorer) PutTableRow(w http.ResponseWriter, r *http.Request) {

}

func (srv *DBExplorer) DeleteTableRow(w http.ResponseWriter, r *http.Request) {

}
