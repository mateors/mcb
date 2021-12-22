package mcb

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/elliotchance/orderedmap"
)

type metrics struct {
	ElapsedTime   string `json:"elapsedTime"`
	ExecutionTime string `json:"executionTime"`
	ResultCount   int    `json:"resultCount"`
	ResultSize    int    `json:"resultSize"`
	ErrorCount    int    `json:"errorCount"`
}

type errorMsg struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

type ResponseMessage struct {
	RequestID string        `json:"requestID"`
	Result    []interface{} `json:"results"`
	Errors    []errorMsg    `json:"errors"`
	Status    string        `json:"status"`
	Metrics   metrics       `json:"metrics"`
}

type nqlQuery struct {
	Statement string `json:"statement"`
	Pretty    bool   `json:"pretty,omitempty"`
}

//DB is a database handle
type DB struct {
	host     string
	port     string
	url      string
	username string
	password string
	bucket   string
}

//Connect method
func Connect(host, userName, passWord, bucketName string, secureConnection bool) *DB {

	var db = &DB{}
	var url string
	if secureConnection {
		url = fmt.Sprintf("https://%s:18093/query/service", host)
		db.port = "18093"
	} else {
		url = fmt.Sprintf("http://%s:8093/query/service", host)
		db.port = "8093"
	}
	//db = &DB{host: host, port: defaultPort, url: url, username: userName, password: passWord, bucket: bucketName}
	db.host = host
	db.url = url
	db.username = userName
	db.password = passWord
	db.bucket = bucketName
	return db
}

//Ping checking couchbase database connection status
func (db *DB) Ping() (string, error) {

	var response string
	timeout := time.Second * 3
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(db.host, db.port), timeout)
	if err != nil {
		response = fmt.Sprintf("Connection error %v", err.Error())
		return response, err
	}
	if conn != nil {
		defer conn.Close()
		response = fmt.Sprintf("Connection successful to %v", net.JoinHostPort(db.host, db.port))
	}
	return response, nil
}

func (db *DB) base64UserPassword() (base64 string) {
	plainTxt := fmt.Sprintf("%s:%s", db.username, db.password)
	base64 = b64.StdEncoding.EncodeToString([]byte(plainTxt))
	return
}

func (db *DB) authorization() (auth string) {
	auth = fmt.Sprintf("Basic %s", db.base64UserPassword())
	return
}

//GetRows no parameter
func (pres *ResponseMessage) GetRows() []map[string]interface{} {

	rows := make([]map[string]interface{}, 0)
	for _, v := range pres.Result {
		ms := v.(map[string]interface{})
		rows = append(rows, ms)
	}
	return rows
}

//GetBucketRows with bucketName
func (pres *ResponseMessage) GetBucketRows(bucketName string) []map[string]interface{} {

	rows := make([]map[string]interface{}, 0)
	for _, v := range pres.Result {

		ms := v.(map[string]interface{})
		if len(bucketName) > 1 {
			rows = append(rows, ms[bucketName].(map[string]interface{}))
		} else {
			rows = append(rows, ms)
		}
	}
	return rows
}

//Query takes an sql statement as input and execute to the couchbase and returns the output
// as pointer to ResponseMessage
func (db *DB) Query(sql string) *ResponseMessage {

	url := db.url
	method := "POST"

	jsonTxt := sqlStatementJSON(sql)
	payload := strings.NewReader(jsonTxt)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		log.Println("ERROR @ Query:", err.Error())
		return nil
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", db.authorization())

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil
	}

	defer res.Body.Close()
	var resPonse ResponseMessage
	json.Unmarshal(body, &resPonse)
	return &resPonse
}

//queryRequest ...
func (db *DB) queryRequest(jsonText string) *ResponseMessage {

	url := db.url
	method := "POST"

	payload := strings.NewReader(jsonText)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Println("ERROR @queryRequest:", err.Error())
		return nil
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", db.authorization())

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer res.Body.Close()

	var resPonse ResponseMessage
	json.Unmarshal(body, &resPonse)
	return &resPonse
}

//ProcessData takes two argument, first argument will come from html form and
//the seconds one is a reference to struct type
func (db *DB) ProcessData(form url.Values, dataFields interface{}) []byte {

	oMap := prepareData(form, dataFields)
	omitList := omitEmptyList(form, dataFields)
	mpRes := make(map[string]interface{})

	// Iterate through all elements from oldest to newest:
	for el := oMap.Front(); el != nil; el = el.Next() {

		if el.Value == nil {

		} else {

			elKey := el.Key.(string)
			elVal := fmt.Sprintf(`%v`, el.Value)     //**correction
			isFound, _ := ArrayFind(omitList, elKey) //check if key exist in omitList

			if isFound && len(elVal) > 0 {
				mpRes[elKey] = el.Value

			} else if !isFound { //100% valid candidate
				mpRes[elKey] = el.Value
			}
		}
	}

	bytes, _ := json.Marshal(mpRes)
	json.Unmarshal(bytes, dataFields) //***s
	return bytes
}

//UpsertIntoBucket ...
func (db *DB) UpsertIntoBucket(docID string, dataFields interface{}) *ResponseMessage {

	bytes, _ := json.Marshal(dataFields)
	upsertQuery := upsertQueryBuilder(db.bucket, docID, string(bytes))
	nqlInsertStatement := sqlStatementJSON(upsertQuery)
	responseMessage := db.queryRequest(nqlInsertStatement)
	return responseMessage
}

//InsertIntoBucket takes 3 argument and returns pointer to ResponseMessage
func (db *DB) InsertIntoBucket(docID string, dataFields interface{}) *ResponseMessage {

	bytes, _ := json.Marshal(dataFields)
	insertQuery := insertQueryBuilder(db.bucket, docID, string(bytes))
	nqlInsertStatement := sqlStatementJSON(insertQuery)
	responseMessage := db.queryRequest(nqlInsertStatement)
	return responseMessage
}

//Insert method for insert, first argument supposed to coming from a html form, second argument
//pass struct field variable as reference placing & as prefix ex: &sVar where sVar is a struct type variable
func (db *DB) Insert(form url.Values, dataFields interface{}) *ResponseMessage {

	bytes := db.ProcessData(form, dataFields)
	docID := form.Get("aid") //docid=aid
	insertQuery := insertQueryBuilder(db.bucket, docID, string(bytes))
	nqlInsertStatement := sqlStatementJSON(insertQuery)

	responseMessage := db.queryRequest(nqlInsertStatement)
	if responseMessage.Status != "success" {
		log.Println(nqlInsertStatement)
	}
	return responseMessage
}

//Upsert method for update and insert both
func (db *DB) Upsert(form url.Values, dataFields interface{}) *ResponseMessage {

	bytes := db.ProcessData(form, dataFields)
	docID := form.Get("aid") //docid=aid

	insertQuery := upsertQueryBuilder(db.bucket, docID, string(bytes))
	nqlInsertStatement := sqlStatementJSON(insertQuery)
	responseMessage := db.queryRequest(nqlInsertStatement)
	return responseMessage
}

func insertQueryBuilder(bucketName, docID, bytesStr string) (nqlStatement string) {

	qs := `INSERT INTO %s (KEY, VALUE)
	VALUES ("%s", %s)
	RETURNING *`
	nqlStatement = fmt.Sprintf(qs, bucketName, docID, bytesStr)
	return
}

func upsertQueryBuilder(bucketName, docID, bytesStr string) (nqlStatement string) {

	qs := `UPSERT INTO %s (KEY, VALUE)
	VALUES ("%s", %s)
	RETURNING *`
	nqlStatement = fmt.Sprintf(qs, bucketName, docID, bytesStr)
	return
}

//Struct fields can be accessed through a struct pointer.
func prepareData(form url.Values, dataFields interface{}) *orderedmap.OrderedMap {

	roMap := orderedmap.NewOrderedMap() //return ordered map
	dtype := reflect.TypeOf(dataFields).Kind().String()

	var typeSlice []string
	if dtype == "ptr" {
		typeSlice = readSructColumnsType(dataFields)

	} else if dtype == "slice" {
		typeSlice = dataFields.([]string)
	}

	oMap := keyValOrder(form, dataFields)
	for i, key := range oMap.Keys() {
		value, _ := oMap.Get(key)
		vtype := typeSlice[i]
		var keyValue string = fmt.Sprintf("%v", value)

		if vtype == "int" {

			kValue, _ := strconv.Atoi(keyValue)
			roMap.Set(key, kValue)

		} else if vtype == "int64" {

			kValue, _ := strconv.ParseInt(keyValue, 10, 64)
			roMap.Set(key, kValue)

		} else if vtype == "float64" {

			kValue, _ := strconv.ParseFloat(keyValue, 64)
			roMap.Set(key, kValue)

		} else if vtype == "slice" {

			roMap.Set(key, form[key.(string)])

		} else {
			roMap.Set(key, value.(string))
		}
	}
	return roMap
}

func readSructColumnsType(i interface{}) []string {

	cols := make([]string, 0)
	iVal := reflect.ValueOf(i).Elem()
	for i := 0; i < iVal.NumField(); i++ {
		f := iVal.Field(i)
		vtype := f.Kind().String()
		cols = append(cols, vtype)
	}
	return cols
}

func omitEmptyList(form url.Values, dataFields interface{}) []string {

	var fieldList []string
	dtype := reflect.TypeOf(dataFields).Kind().String()

	if dtype == "ptr" {

		iVal := reflect.ValueOf(dataFields).Elem()
		typ := iVal.Type()

		for i := 0; i < iVal.NumField(); i++ {

			tag := typ.Field(i).Tag.Get("json")
			var omitFound bool
			if strings.Contains(tag, ",") {
				omitFound = true
			}

			if omitFound && len(form.Get(tag)) == 0 {
				commaFoundAt := strings.Index(tag, ",")
				ntag := tag[0:commaFoundAt]
				fieldList = append(fieldList, ntag)
			}
		}

	} else if dtype == "slice" {
		fieldList = dataFields.([]string)
	}
	return fieldList
}

//KeyValOrder takes two argument and returns pointer to an orderedMap
func keyValOrder(form url.Values, dataFields interface{}) *orderedmap.OrderedMap {

	oMap := orderedmap.NewOrderedMap()
	dtype := reflect.TypeOf(dataFields).Kind().String()

	if dtype == "ptr" {

		iVal := reflect.ValueOf(dataFields).Elem()
		typ := iVal.Type()

		for i := 0; i < iVal.NumField(); i++ {

			tag := typ.Field(i).Tag.Get("json")
			var omitFound bool
			if strings.Contains(tag, ",") {
				omitFound = true
				commaFoundAt := strings.Index(tag, ",")
				tag = tag[0:commaFoundAt]
			}
			if omitFound && len(form.Get(tag)) == 0 {
				oMap.Set(tag, "")
			} else {
				oMap.Set(tag, form.Get(tag))
			}
		}

	} else if dtype == "slice" {

		for _, tag := range dataFields.([]string) {
			oMap.Set(tag, form.Get(tag))
		}
	}
	return oMap
}

func sqlStatementJSON(sql string) string {

	nqlObj := new(nqlQuery)
	nqlObj.Statement = sql //fmt.Sprintf(`SELECT * FROM master_erp WHERE type="%v"`, "login_session")
	rbytes, _ := json.Marshal(nqlObj)
	return string(rbytes)
}

//ReturnIndexByValue to Get index number by its value from a slice
func ReturnIndexByValue(s []string, val string) (index int) {

	for index, v := range s {
		if v == val {
			return index
		}
	}
	return -1
}

//ArrayFind Find a value in_array with its index number
func ArrayFind(array []string, value string) (bool, int) {

	indx := ReturnIndexByValue(array, value)
	if indx == -1 {
		return false, -1
	}
	return true, indx
}
