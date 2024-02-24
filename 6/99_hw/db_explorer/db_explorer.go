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
	"sort"
	"strconv"
	"strings"
)

type HttpResponse struct {
	Error    string `json:"error,omitempty"`
	Response any    `json:"response,omitempty"`
}

func (r HttpResponse) Write(w http.ResponseWriter, status int) {
	result, _ := json.Marshal(r)
	w.WriteHeader(status)
	w.Write(result)
}

func NewErrorResponse(err error) HttpResponse {
	return HttpResponse{Error: err.Error()}
}

func NewResponse(v any) HttpResponse {
	return HttpResponse{Response: v}
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

type ErrInvalidFieldType string

func (e ErrInvalidFieldType) Error() string {
	return fmt.Sprintf("field %s have invalid type", string(e))
}

var (
	ErrTableNotFound  = errors.New("unknown table")
	ErrRecordNotFound = errors.New("record not found")
)

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

func (s *MySQLDBExplorerService) GetTables() []string {
	tables := generics.MapKeys(s.tablesColumns)
	sort.Strings(tables)
	return tables
}

func (s *MySQLDBExplorerService) GetTableRows(ctx context.Context, table string, limit, offset int) ([]map[string]any, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return nil, ErrTableNotFound
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
		return nil, ErrTableNotFound
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}

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
		return nil, ErrTableNotFound
	}

	allowedFields, err := filterFieldsForCreate(columns, data)
	if err != nil {
		return nil, err
	}

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

	primaryKeyColumn, ok := generics.FindFirst(columns, func(x Column) bool {
		return x.Key == "PRI"
	})
	if !ok {
		return nil, errors.New("primary key column is not found")
	}

	result := map[string]any{
		primaryKeyColumn.Field: insertedId,
	}
	return result, nil
}

func (s *MySQLDBExplorerService) UpdateTableRow(ctx context.Context, table string, key any, data map[string]any) (int64, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return 0, ErrTableNotFound
	}

	primaryKeyColumn, ok := generics.FindFirst(columns, func(x Column) bool {
		return x.Key == "PRI"
	})
	if !ok {
		return 0, errors.New("primary key column is not found")
	}
	allowedFields, err := filterFieldsForUpdate(columns, data)
	if err != nil {
		return 0, err
	}

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
		return 0, err
	}

	affectedRows, err := sqlResult.RowsAffected()
	return affectedRows, err
}

func (s *MySQLDBExplorerService) DeleteTableRow(ctx context.Context, table string, key any) (int64, error) {
	columns, tableExists := s.tablesColumns[table]
	if !tableExists {
		return 0, ErrTableNotFound
	}

	primaryKeyColumn, ok := generics.FindFirst(columns, func(x Column) bool {
		return x.Key == "PRI"
	})
	if !ok {
		return 0, errors.New("couldn't find primary key field")
	}
	// At the moment we checked if table is valid with looking up in s.tablesColumns[table].
	// So there can't be SQL-injection in the table variable.
	// Analogically with primaryKeyColumn.Field.
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, primaryKeyColumn.Field)
	sqlResult, err := s.db.ExecContext(ctx, query, key)
	if err != nil {
		return 0, err
	}

	rowsDeleted, err := sqlResult.RowsAffected()
	return rowsDeleted, err
}

func filterFieldsForUpdate(columns []Column, data map[string]any) ([]string, error) {
	allowedFields := make([]string, 0, len(data))
DataLoop:
	for dataKey, dataValue := range data {
		for _, column := range columns {
			if dataKey == column.Field {
				// column is found
				if column.Key == "PRI" || column.Extra == "auto_increment" {
					// can't change primary keys & fields with auto increment
					return nil, ErrInvalidFieldType(dataKey)
				}

				switch dataValue.(type) {
				case string:
					switch column.Type {
					case "varchar(255)", "text":
					default:
						return nil, ErrInvalidFieldType(dataKey)
					}
				case int:
					switch column.Type {
					case "int":
					default:
						return nil, ErrInvalidFieldType(dataKey)
					}
				case float64:
					switch column.Type {
					case "float":
					default:
						return nil, ErrInvalidFieldType(dataKey)
					}
				}

				if dataValue == nil && column.Null == "NO" {
					return nil, ErrInvalidFieldType(dataKey)
				}

				//it's good field
				allowedFields = append(allowedFields, dataKey)
				continue DataLoop
			}
		}
		// column is not found => skip it
	}
	return allowedFields, nil
}

func filterFieldsForCreate(columns []Column, data map[string]any) ([]string, error) {
	allowedFields := make([]string, 0, len(data))
DataLoop:
	for dataKey, dataValue := range data {
		for _, column := range columns {
			if dataKey == column.Field {
				// column is found
				if column.Key == "PRI" || column.Extra == "auto_increment" {
					continue DataLoop
				}

				switch dataValue.(type) {
				case string:
					switch column.Type {
					case "varchar(255)", "text":
					default:
						return nil, ErrInvalidFieldType(dataKey)
					}
				case int:
					switch column.Type {
					case "int":
					default:
						return nil, ErrInvalidFieldType(dataKey)
					}
				case float64:
					switch column.Type {
					case "float":
					default:
						return nil, ErrInvalidFieldType(dataKey)
					}
				}

				if dataValue == nil && column.Null == "NO" {
					return nil, ErrInvalidFieldType(dataKey)
				}

				//it's good field
				allowedFields = append(allowedFields, dataKey)
				continue DataLoop
			}
		}
		// column is not found => skip it
	}
	return allowedFields, nil
}

type DBExplorerService interface {
	GetTables() []string
	GetTableRows(ctx context.Context, table string, limit, offset int) ([]map[string]any, error)
	GetTableRow(ctx context.Context, table string, key any) (map[string]any, error)
	CreateTableRow(ctx context.Context, table string, data map[string]any) (map[string]any, error)
	UpdateTableRow(ctx context.Context, table string, key any, data map[string]any) (int64, error)
	DeleteTableRow(ctx context.Context, table string, key any) (int64, error)
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

func (srv *DBExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", srv.GetAllTables)
	mux.HandleFunc("GET /{table}", srv.GetTableRows)
	mux.HandleFunc("PUT /{table}/", srv.PutTableRow)
	mux.HandleFunc("GET /{table}/{id}", srv.GetTableRow)
	mux.HandleFunc("POST /{table}/{id}", srv.PostTableRow)
	mux.HandleFunc("DELETE /{table}/{id}", srv.DeleteTableRow)

	mux.ServeHTTP(w, r)
}

func (srv *DBExplorer) GetAllTables(w http.ResponseWriter, _ *http.Request) {
	NewResponse(map[string]any{
		"tables": srv.service.GetTables(),
	}).Write(w, http.StatusOK)
}

func (srv *DBExplorer) GetTableRows(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	if table == "" {
		NewErrorResponse(ErrTableNotFound).Write(w, http.StatusNotFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}
	limitRaw, offsetRaw := r.Form.Get("limit"), r.Form.Get("offset")
	limit, err := strconv.Atoi(limitRaw)
	if err != nil {
		limit = 5
	}
	offset, err := strconv.Atoi(offsetRaw)
	if err != nil {
		offset = 0
	}

	rows, err := srv.service.GetTableRows(r.Context(), table, limit, offset)
	if err != nil {
		if errors.Is(err, ErrTableNotFound) {
			NewErrorResponse(err).Write(w, http.StatusNotFound)
			return
		}
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	NewResponse(map[string]any{
		"records": rows,
	}).Write(w, http.StatusOK)
}

func (srv *DBExplorer) GetTableRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	if table == "" {
		NewErrorResponse(ErrTableNotFound).Write(w, http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		NewErrorResponse(errors.New("unknown id")).Write(w, http.StatusNotFound)
		return
	}

	row, err := srv.service.GetTableRow(r.Context(), table, id)
	if err != nil {
		if errors.Is(err, ErrTableNotFound) {
			NewErrorResponse(err).Write(w, http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrRecordNotFound) {
			NewErrorResponse(err).Write(w, http.StatusNotFound)
			return
		}
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	NewResponse(map[string]any{
		"record": row,
	}).Write(w, http.StatusOK)
}

func (srv *DBExplorer) PutTableRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	if table == "" {
		NewErrorResponse(ErrTableNotFound).Write(w, http.StatusNotFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusBadRequest)
		return
	}

	var body map[string]any
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	result, err := srv.service.CreateTableRow(r.Context(), table, body)
	if err != nil {
		if errors.Is(err, ErrTableNotFound) {
			NewErrorResponse(err).Write(w, http.StatusNotFound)
			return
		}
		var invalidFieldErr ErrInvalidFieldType
		if errors.As(err, &invalidFieldErr) {
			NewErrorResponse(err).Write(w, http.StatusBadRequest)
			return
		}
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	NewResponse(result).Write(w, http.StatusOK)
}

func (srv *DBExplorer) PostTableRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	if table == "" {
		NewErrorResponse(ErrTableNotFound).Write(w, http.StatusNotFound)
		return
	}

	idRaw := r.PathValue("id")
	if idRaw == "" {
		NewErrorResponse(errors.New("unknown id")).Write(w, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusBadRequest)
		return
	}

	var body map[string]any
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	updatedRows, err := srv.service.UpdateTableRow(r.Context(), table, id, body)
	if err != nil {
		if errors.Is(err, ErrTableNotFound) {
			NewErrorResponse(err).Write(w, http.StatusNotFound)
			return
		}

		var invalidFieldErr ErrInvalidFieldType
		if errors.As(err, &invalidFieldErr) {
			NewErrorResponse(err).Write(w, http.StatusBadRequest)
			return
		}

		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	NewResponse(map[string]any{
		"updated": updatedRows,
	}).Write(w, http.StatusOK)
}

func (srv *DBExplorer) DeleteTableRow(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	if table == "" {
		NewErrorResponse(ErrTableNotFound).Write(w, http.StatusNotFound)
		return
	}

	idRaw := r.PathValue("id")
	if idRaw == "" {
		NewErrorResponse(errors.New("unknown id")).Write(w, http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idRaw)
	if err != nil {
		NewErrorResponse(err).Write(w, http.StatusBadRequest)
		return
	}

	deletedRows, err := srv.service.DeleteTableRow(r.Context(), table, id)
	if err != nil {
		if errors.Is(err, ErrTableNotFound) {
			NewErrorResponse(err).Write(w, http.StatusNotFound)
			return
		}
		NewErrorResponse(err).Write(w, http.StatusInternalServerError)
		return
	}

	NewResponse(map[string]any{
		"deleted": deletedRows,
	}).Write(w, http.StatusOK)
}
