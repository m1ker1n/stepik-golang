package main

import (
	"cmp"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const datasetPath = "dataset.xml"

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

type Row struct {
	Id             int    `xml:"id"`
	Guid           string `xml:"guid"`
	IsActive       bool   `xml:"isActive"`
	Balance        string `xml:"balance"`
	PictureUrl     string `xml:"picture"`
	Age            int    `xml:"age"`
	EyeColor       string `xml:"eyeColor"`
	FirstName      string `xml:"first_name"`
	LastName       string `xml:"last_name"`
	Gender         string `xml:"gender"`
	Company        string `xml:"company"`
	Email          string `xml:"email"`
	Phone          string `xml:"phone"`
	Address        string `xml:"address"`
	About          string `xml:"about"`
	Registered     string `xml:"registered"`
	FavouriteFruit string `xml:"favourite_fruit"`
}

func (r Row) ToUser() User {
	return User{
		Id:     r.Id,
		Name:   r.FirstName + " " + r.LastName,
		Age:    r.Age,
		About:  r.About,
		Gender: r.Gender,
	}
}

func SearchServerBadResultJson(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`[{"Id":15,"Name":"Allison Valdez","Age":21,"About":"Labore excepteur voluptate velit occaecat est nisi minim. Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. Ullamco sint nisi voluptate cillum nostrud aliquip et minim. Enim duis esse do aute qui officia ipsum ut occaecat deserunt`))
}

func SearchServerBadErrorJson(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(`{"Error": "ErrorBadOrderField"`))
}

func SearchServerInternalError(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func SearchServerNonStopRedirection(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	accessToken := r.Header.Get("AccessToken")
	if accessToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	params, err := receiveSearchRequest(r)
	if err != nil {
		response := SearchErrorResponse{err.Error()}
		marshalledResponse, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(marshalledResponse)
		return
	}

	file, err := os.ReadFile(datasetPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var dataset Root
	err = xml.Unmarshal(file, &dataset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	users := func(rows []Row) []User {
		res := make([]User, len(rows))
		for i := range rows {
			res[i] = rows[i].ToUser()
		}
		return res
	}(dataset.Rows)

	results, err := searchServer(
		users,
		params.Query,
		params.OrderField,
		params.OrderBy,
		params.Limit,
		params.Offset,
	)
	if err != nil {
		response := SearchErrorResponse{err.Error()}
		marshalledResponse, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(marshalledResponse)
		return
	}

	marshalledResults, err := json.Marshal(results)
	if err != nil {
		panic(fmt.Errorf("cannot pack results into json: %w", err))
	}

	_, _ = w.Write(marshalledResults)
}

func receiveSearchRequest(r *http.Request) (SearchRequest, error) {
	errs := make([]error, 0, 3)

	query := r.FormValue("query")
	orderField := r.FormValue("order_field")
	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		errs = append(errs, fmt.Errorf("could not parse order_by: %w", err))
	}
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		// cannot parse to int
		errs = append(errs, fmt.Errorf("could not parse limit: %w", err))
	}
	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		// cannot parse to int
		errs = append(errs, fmt.Errorf("could not parse offset: %w", err))
	}

	var resErr error = nil
	if len(errs) != 0 {
		errsStr := ""
		for _, err = range errs {
			errsStr += err.Error() + ";"
		}
		resErr = errors.New(errsStr)
	}

	return SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderField,
		OrderBy:    orderBy,
	}, resErr
}

// searchServer creates its own slice of users
// so, it's ok to use results of function without worrying it'll change
// incoming data
func searchServer(users []User, query string, orderField string, orderBy int, limit int, offset int) ([]User, error) {
	result := filter(users, containsQueryInNameOrAbout(query))

	cmpFunc, err := cmpUser(orderField, orderBy)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(result, cmpFunc)

	return withOffsetAndLimit(result, limit, offset)
}

// filter creates a new slice
func filter[T any](elems []T, predicate func(T) bool) []T {
	res := make([]T, 0, len(elems))
	for _, el := range elems {
		if predicate(el) {
			res = append(res, el)
		}
	}
	return res
}

// containsQueryInNameOrAbout creates a predicate
// for checking if user contains query in Name or About field
func containsQueryInNameOrAbout(query string) func(u User) bool {
	return func(u User) bool {
		return strings.Contains(u.Name, query) || strings.Contains(u.About, query)
	}
}

// cmpUser creates a function for sorting slice of users
// if orderField, orderBy are incorrect returns error
func cmpUser(orderField string, orderBy int) (func(a, b User) int, error) {
	var cmpFunc func(a, b User) int
	switch orderField {
	case "Id":
		cmpFunc = func(a, b User) int {
			return cmp.Compare(a.Id, b.Id)
		}
	case "", "Name":
		cmpFunc = func(a, b User) int {
			return cmp.Compare(a.Name, b.Name)
		}
	case "Age":
		cmpFunc = func(a, b User) int {
			return cmp.Compare(a.Age, b.Age)
		}
	default:
		return nil, errors.New("ErrorBadOrderField")
	}

	switch orderBy {
	case OrderByAsc, OrderByAsIs, OrderByDesc:
	default:
		return nil, errors.New("bad orderBy value, must be -1, 0 or 1")
	}

	return func(a, b User) int {
		//-orderBy because orderByAsc = -1 and sortFunc sorts by asc by default
		//so -orderBy = 1, and it keeps ascending order
		return -orderBy * cmpFunc(a, b)
	}, nil
}

func withOffsetAndLimit[T any](elems []T, limit, offset int) ([]T, error) {
	if offset < 0 {
		return nil, errors.New("offset must be >= 0")
	}
	if offset > len(elems) {
		return nil, errors.New("offset must be <= len(elems)")
	}
	if limit < 0 {
		return nil, errors.New("limit must be >= 0")
	}
	if limit+offset <= len(elems) {
		return elems[offset : offset+limit], nil
	} else {
		return elems[offset:], nil
	}
}

func TestSearchServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	tsTimeout := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		SearchServer(w, r)
	}))
	defer tsTimeout.Close()

	tsBadResultJson := httptest.NewServer(http.HandlerFunc(SearchServerBadResultJson))
	defer tsBadResultJson.Close()

	tsBadErrorJson := httptest.NewServer(http.HandlerFunc(SearchServerBadErrorJson))
	defer tsBadErrorJson.Close()

	tsInternalError := httptest.NewServer(http.HandlerFunc(SearchServerInternalError))
	defer tsInternalError.Close()

	tsNonStopRedirect := httptest.NewServer(http.HandlerFunc(SearchServerNonStopRedirection))
	defer tsNonStopRedirect.Close()

	type testCase struct {
		name string

		server *httptest.Server

		clientAccessToken string

		request SearchRequest

		expectError bool
	}

	const (
		validAccessToken   = "kek"
		invalidAccessToken = ""
	)

	var (
		validReq = SearchRequest{
			Limit:      100,
			Offset:     0,
			Query:      "",
			OrderField: "",
			OrderBy:    0,
		}
		validReqLowLimit = SearchRequest{
			Limit:      1,
			Offset:     0,
			Query:      "",
			OrderField: "",
			OrderBy:    0,
		}
		validSingleResult = SearchRequest{
			Limit:      1,
			Offset:     0,
			Query:      "Boyd Wolf",
			OrderField: "",
			OrderBy:    0,
		}
	)

	tests := []testCase{
		{
			name:              "client error trying to send request: server redirects constantly",
			server:            tsNonStopRedirect,
			clientAccessToken: "",
			request:           SearchRequest{},
			expectError:       true,
		},
		{
			name:              "timeout",
			server:            tsTimeout,
			clientAccessToken: validAccessToken,
			request:           validReq,
			expectError:       true,
		},
		{
			name:              "invalid access token",
			server:            ts,
			clientAccessToken: invalidAccessToken,
			request:           validReq,
			expectError:       true,
		},
		{
			name:              "ok",
			server:            ts,
			clientAccessToken: validAccessToken,
			request:           validReq,
			expectError:       false,
		},
		{
			name:              "ok low limit",
			server:            ts,
			clientAccessToken: validAccessToken,
			request:           validReqLowLimit,
			expectError:       false,
		},
		{
			name:              "ok no next page",
			server:            ts,
			clientAccessToken: validAccessToken,
			request:           validSingleResult,
			expectError:       false,
		},
		{
			name:              "invalid limit",
			server:            ts,
			clientAccessToken: validAccessToken,
			request: SearchRequest{
				Limit: -1,
			},
			expectError: true,
		},
		{
			name:              "invalid offset",
			server:            ts,
			clientAccessToken: validAccessToken,
			request: SearchRequest{
				Offset: -1,
			},
			expectError: true,
		},
		{
			name:              "invalid order field",
			server:            ts,
			clientAccessToken: validAccessToken,
			request: SearchRequest{
				OrderField: "wtf",
			},
			expectError: true,
		},
		{
			name:              "bad result json",
			server:            tsBadResultJson,
			clientAccessToken: validAccessToken,
			request:           validReq,
			expectError:       true,
		},
		{
			name:              "bad error json",
			server:            tsBadErrorJson,
			clientAccessToken: validAccessToken,
			request:           validReq,
			expectError:       true,
		},
		{
			name:              "internal error",
			server:            tsInternalError,
			clientAccessToken: validAccessToken,
			request:           validReq,
			expectError:       true,
		},
		{
			name:              "unknown bad request error: bad server",
			server:            ts,
			clientAccessToken: validAccessToken,
			request: SearchRequest{
				OrderBy: 10,
			},
			expectError: true,
		},
	}

	for i, tt := range tests {
		client := SearchClient{
			AccessToken: tt.clientAccessToken,
			URL:         tt.server.URL,
		}

		_, err := client.FindUsers(tt.request)

		if tt.expectError && err == nil {
			t.Errorf("[%d] [%s] expected error, got nil", i, tt.name)
		}

		if !tt.expectError && err != nil {
			t.Errorf("[%d] [%s] did not expect error, got %s", i, tt.name, err)
		}
	}

}
