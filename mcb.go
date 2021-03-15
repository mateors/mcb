package mcb

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mateors/mtool"

	"github.com/elliotchance/orderedmap"
)

type metrics struct {
	ElapsedTime   string `json:"elapsedTime"`
	ExecutionTime string `json:"executionTime"`
	ResultCount   int    `json:"resultCount"`
	ResultSize    int    `json:"resultSize"`
	ErrorCount    int    `json:"errorCount"`
}

//Struct First character must be a capital letter
type errorMsg struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

//ResponseMessage Main struct
type ResponseMessage struct {
	RequestID string `json:"requestID"`
	//Result    []string   `json:"results"`
	Result  []interface{} `json:"results"`
	Errors  []errorMsg    `json:"errors"`
	Status  string        `json:"status"`
	Metrics metrics       `json:"metrics"`
}

type nqlQuery struct {
	Statement string `json:"statement"`
	Pretty    bool   `json:"pretty,omitempty"`
	//Metrics   bool   `json:"metrics"`
}

//DB is a database handle
type DB struct {
	host     string
	url      string
	username string
	password string
	bucket   string
}

//Connect method
func Connect(host, userName, passWord, bucketName string) *DB {

	url := fmt.Sprintf("http://%s:8093/query/service", host)
	db := &DB{host: host, url: url, username: userName, password: passWord, bucket: bucketName}
	return db
}

//Ping checking couchbase database connection status
func (db *DB) Ping() (string, error) {

	var response string
	timeout := time.Second * 3
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(db.host, "8093"), timeout)
	if err != nil {
		response = fmt.Sprintf("Connection error %v", err.Error())
		return response, err
	}
	if conn != nil {
		defer conn.Close()
		response = fmt.Sprintf("Connection successful to %v", net.JoinHostPort(db.host, "8093"))
	}

	return response, nil
}

func (db *DB) base64UserPassword() (base64 string) {

	plainTxt := fmt.Sprintf("%s:%s", db.username, db.password)
	base64 = b64.StdEncoding.EncodeToString([]byte(plainTxt))
	return
}

func (db *DB) authorization() (auth string) {
	//"Basic QWRtaW5pc3RyYXRvcjpNb3N0YWluMzIxJA=="
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

//GetBucketRows ...
func (pres *ResponseMessage) GetBucketRows(bucketName string) []map[string]interface{} {

	rows := make([]map[string]interface{}, 0)

	for _, v := range pres.Result {

		ms := v.(map[string]interface{})
		//fmt.Println(i, "==>", ms["master_erp"])
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
	//payload := strings.NewReader("{\n \"statement\": \"SELECT * FROM master_erp WHERE type='login_session'\"\n}\n\n")

	payload := strings.NewReader(jsonTxt)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println("ERROR @ Query:", err.Error())
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", db.authorization())

	res, err := client.Do(req)
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	//local variable as a pointer
	var resPonse ResponseMessage

	json.Unmarshal(body, &resPonse)

	return &resPonse
}

//queryRequest ...
func (db *DB) queryRequest(jsonText string) *ResponseMessage {

	url := db.url
	method := "POST"

	//payload := strings.NewReader("{\n \"statement\": \"INSERT INTO master_erp (KEY, VALUE) VALUES (\\\"login_session::104\\\", { \\\"type\\\": \\\"login_session\\\", \\\"cid\\\":1,\\\"device_info\\\":\\\"device_log::2\\\",\\\"session_code\\\":\\\"000-1111-2222-333-4444\\\",\\\"login_id\\\":1,\\\"ip_address\\\":\\\"0.0.0.0\\\",\\\"city\\\":\\\"Dhaka\\\",\\\"country\\\":\\\"Bangladesh\\\",\\\"login_time\\\":\\\"2020-06-11 10:30:00\\\",\\\"create_date\\\":\\\"2020-06-11 09:00:30\\\",\\\"status\\\": 1 }) RETURNING *\"\n}\n\n")
	payload := strings.NewReader(jsonText)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println("ERROR @queryRequest:", err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", db.authorization())

	res, err := client.Do(req)
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	//local variable as a pointer
	var resPonse ResponseMessage

	json.Unmarshal(body, &resPonse)

	//resPonse.Errors[0].Code
	//resPonse.Status

	// fmt.Println(string(body))
	// fmt.Println("resPonse::>>", resPonse)
	// bytes, _ := json.Marshal(resPonse)
	// fmt.Println()
	// fmt.Println()
	// fmt.Println(string(bytes))

	return &resPonse
}

//ProcessData takes two argument, first argument will come from html form and
//the seconds one is a reference to struct type
func (db *DB) ProcessData(form url.Values, dataFields interface{}) []byte {

	oMap := prepareData(form, dataFields)
	//fmt.Println("oMap:>", oMap)
	omitList := omitEmptyList(form, dataFields)
	//fmt.Println("OmitFieldList:", omitList)

	mpRes := make(map[string]interface{}, 0)

	// Iterate through all elements from oldest to newest:
	for el := oMap.Front(); el != nil; el = el.Next() {

		//fmt.Println(el.Key, "===", el.Value)
		//fmt.Printf("%v %T\n", el.Value, el.Value)
		if el.Value == nil { //|| el.Value == ""
			//fmt.Println("nil value for", el.Key)
			//mpRes[el.Key.(string)] = nil

		} else {

			elKey := el.Key.(string)
			elVal := fmt.Sprintf(`%v`, el.Value)           //**correction
			isFound, _ := mtool.ArrayFind(omitList, elKey) //check if key exist in omitList

			if isFound == true && len(elVal) > 0 {
				//fmt.Println(elKey, "==>", elVal, len(elVal))
				mpRes[elKey] = el.Value

			} else if isFound == false { //100% valid candidate
				mpRes[elKey] = el.Value

			} else {
				//fmt.Println("###### ProcessDataOmit::", elKey, el.Value)
			}
		}

	}

	bytes, _ := json.Marshal(mpRes)
	//fmt.Println("bytes:", string(bytes))

	//fmt.Println()
	//var logSessData2 models.LoginSession
	json.Unmarshal(bytes, dataFields) //***s
	//bytes2, _ := json.Marshal(intrfc)

	return bytes

}

//func EncodeBase64(plainText string) (base64 string) {
//import b64 "encoding/base64"
//base64 = b64.StdEncoding.EncodeToString([]byte(plainText))
//fmt.Println(base64)
//return
//}

//UpsertIntoBucket ...
func (db *DB) UpsertIntoBucket(docID, bucketName string, dataFields interface{}) *ResponseMessage {

	bytes, _ := json.Marshal(dataFields)
	//upsertQueryBuilder()
	upsertQuery := db.upsertQueryBuilder(docID, string(bytes))
	nqlInsertStatement := sqlStatementJSON(upsertQuery)
	responseMessage := db.queryRequest(nqlInsertStatement)
	return responseMessage
}

//InsertIntoBucket takes 3 argument and returns pointer to ResponseMessage
func (db *DB) InsertIntoBucket(docID, bucketName string, dataFields interface{}) *ResponseMessage {

	bytes, _ := json.Marshal(dataFields)
	insertQuery := db.insertQueryBuilder(docID, string(bytes))
	nqlInsertStatement := sqlStatementJSON(insertQuery)
	responseMessage := db.queryRequest(nqlInsertStatement)

	return responseMessage
}

//Insert method for insert, first argument supposed to coming from a html form, second argument
//pass struct field variable as reference placing & as prefix ex: &sVar where sVar is a struct type variable
func (db *DB) Insert(form url.Values, dataFields interface{}) *ResponseMessage {

	//bucketName := "master_erp"
	//docID := "12121"
	bytes := db.ProcessData(form, dataFields)
	//db.ProcessData(form, intrfc)
	//json.Unmarshal(bytes, intrfc) //***s

	//fmt.Println("DATA>>>>", intrfc)
	//bytes, _ := json.Marshal(intrfc)
	//fmt.Println("intrfcBytes:", string(bytes))

	docID := form.Get("aid") //docid=aid

	//json.Unmarshal(bytes, intrfc)
	insertQuery := db.insertQueryBuilder(docID, string(bytes))
	//insertQuery := insertQueryBuilder(bucketName, docID, intrfc)

	//fmt.Println(insertQuery)
	nqlInsertStatement := sqlStatementJSON(insertQuery)
	//fmt.Println()
	//fmt.Println(nqlInsertStatement, form)

	responseMessage := db.queryRequest(nqlInsertStatement)
	//fmt.Println(responseMessage.Status)
	if responseMessage.Status != "success" {
		fmt.Println(nqlInsertStatement)
	}

	return responseMessage
}

//Upsert method for update and insert both
func (db *DB) Upsert(form url.Values, dataFields interface{}) *ResponseMessage {

	//bucketName := "master_erp"
	//docID := "12121"
	bytes := db.ProcessData(form, dataFields)
	//db.ProcessData(form, intrfc)
	//json.Unmarshal(bytes, intrfc) //***s

	//fmt.Println("DATA>>>>", intrfc)

	//bytes, _ := json.Marshal(intrfc)
	//fmt.Println("intrfcBytes:", string(bytes))

	docID := form.Get("aid") //docid=aid

	insertQuery := db.upsertQueryBuilder(docID, string(bytes))
	//insertQuery := insertQueryBuilder(bucketName, docID, intrfc)

	//fmt.Println(insertQuery)
	nqlInsertStatement := sqlStatementJSON(insertQuery)
	responseMessage := db.queryRequest(nqlInsertStatement)

	return responseMessage
}

func (db *DB) insertQueryBuilder(docID, bytesStr string) (nqlStatement string) {

	//docID := fmt.Sprintf("%s::%v", tableName, totalDocs)
	//UPSERT
	qs := `INSERT INTO %s (KEY, VALUE)
	VALUES ("%s", %s)
	RETURNING *`

	nqlStatement = fmt.Sprintf(qs, db.bucket, docID, bytesStr)

	return

}

func (db *DB) upsertQueryBuilder(docID, bytesStr string) (nqlStatement string) {

	qs := `UPSERT INTO %s (KEY, VALUE)
	VALUES ("%s", %s)
	RETURNING *`
	nqlStatement = fmt.Sprintf(qs, db.bucket, docID, bytesStr)
	return
}

//Struct fields can be accessed through a struct pointer.

func prepareData(form url.Values, dataFields interface{}) *orderedmap.OrderedMap {

	//uMap := make(map[string]interface{}, 0)
	roMap := orderedmap.NewOrderedMap() //return ordered map

	typeSlice := readSructColumnsType(dataFields)
	oMap := keyValOrder(form, dataFields)

	for i, key := range oMap.Keys() {
		value, _ := oMap.Get(key)
		vtype := typeSlice[i]
		//fmt.Println(key, "==", value, "->", vtype)
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

	//cols := []string{}
	cols := make([]string, 0)
	iVal := reflect.ValueOf(i).Elem()
	//typ := iVal.Type()
	//fmt.Printf("typ: %v", typ)
	for i := 0; i < iVal.NumField(); i++ {

		f := iVal.Field(i)
		// tag := typ.Field(i).Tag.Get("json")
		// cols = append(cols, tag)

		//f.Interface().(type)
		vtype := f.Kind().String()
		//fmt.Printf(", kind: %v", f.Kind().String())
		cols = append(cols, vtype)

	}

	return cols
}

func omitEmptyList(form url.Values, dataFields interface{}) []string {

	var fieldList []string
	iVal := reflect.ValueOf(dataFields).Elem()
	typ := iVal.Type()

	for i := 0; i < iVal.NumField(); i++ {

		tag := typ.Field(i).Tag.Get("json")
		var omitFound bool
		if strings.Contains(tag, ",") == true {
			omitFound = true
		}

		if omitFound == true && len(form.Get(tag)) == 0 {
			commaFoundAt := strings.Index(tag, ",")
			ntag := tag[0:commaFoundAt]
			fieldList = append(fieldList, ntag)
		}
	}

	return fieldList
}

//KeyValOrder takes two argument and returns pointer to an orderedMap
func keyValOrder(form url.Values, dataFields interface{}) *orderedmap.OrderedMap {

	//uMap := make(map[string]interface{}, 0)
	oMap := orderedmap.NewOrderedMap()
	iVal := reflect.ValueOf(dataFields).Elem()
	typ := iVal.Type()

	for i := 0; i < iVal.NumField(); i++ {

		tag := typ.Field(i).Tag.Get("json")

		var omitFound bool
		if strings.Contains(tag, ",") == true {
			omitFound = true
			commaFoundAt := strings.Index(tag, ",")
			//fmt.Println("commaFoundAt-->", commaFoundAt, tag)
			tag = tag[0:commaFoundAt]
		}

		//ignored omitemty field which has 0 length
		if omitFound == true && len(form.Get(tag)) == 0 {
			oMap.Set(tag, "")
			//fmt.Println(">>", tag, "=", form.Get(tag), omitFound, len(form.Get(tag)))
		} else {
			oMap.Set(tag, form.Get(tag))
		}
		//fmt.Println(">>", tag, "=", form.Get(tag), omitFound, len(form.Get(tag)))
	}

	return oMap
}

//
func sqlStatementJSON(sql string) string {

	nqlObj := new(nqlQuery)
	nqlObj.Statement = sql //fmt.Sprintf(`SELECT * FROM master_erp WHERE type="%v"`, "login_session")
	//nqlObj.Pretty = true
	//nqlObj.Metrics = false

	rbytes, _ := json.Marshal(nqlObj)
	//fmt.Println(string(rbytes))

	return string(rbytes)
}
