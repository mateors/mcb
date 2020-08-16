# mcb
mateors couchbase database library

This library has been written on golang (1.14.4) for couchbase community edition 6.5.1
You can use this library to work with couchbase database, couchbase SDK for golang developers.

## how to install?
> go get github.com/mateors/mcb

## how to use on code?
```
var db *mcb.DB

type myTable struct {
	Name       string   `json:"name"`
	Age        int      `json:"age"`
	Profession string   `json:"profession"`
	Hobbies    []string `json:"hobbies"`
	Type       string   `json:"type"`
}

func init() {

	db = mcb.Connect("<host>", "<username>", "<password>")

	res, err := db.Ping()
	if err != nil {

		fmt.Println(res)
		os.Exit(1)
	}
	fmt.Println(res, err)

}
func main(){


    //How to insert into couchbase bucket
	var myData myTable

	form := make(url.Values, 0)
	form.Add("bucket", "master_erp") //bucket Name
	form.Add("aid", "d006") //document ID
	form.Add("name", "Mostain Billah")
	form.Add("age", "36")
	form.Add("profession", "Developer")
	form.Add("hobbies", "Programming")
	form.Add("hobbies", "Problem Solving")
    form.Add("type", "participant") //what type of data or table name in general (SQL)

	p := db.Insert(form, &myData)
	fmt.Println("Status:", p.Status, form)

    //How to retrieve from couchbase bucket (selected fields only)

    pres := db.Query("SELECT aid,name,age,profession FROM master_erp WHERE type='participant'")
	rows := pres.GetRows()

	fmt.Println("Total Rows:",len(rows))
	fmt.Println(rows)

    //How to retrieve from couchbase bucket (All fields using *)

    pres := db.Query("SELECT * FROM master_erp WHERE type='participant'")
	rows := pres.GetBucketRows("master_erp") //bucketName as argument

	fmt.Println("Total Rows:",len(rows))
	fmt.Println(rows)

}

```
