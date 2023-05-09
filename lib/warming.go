package lib

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"time"
	"fmt"
	// "github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	// "github.com/aws/aws-sdk-go/service/rds"
	// "strings"
)


func WarmingTarget(dbObj *sql.DB,d string,i int) ([]string,error) {
	var dbs []string

	// Get Schema List
	getSchema := `
		SELECT 
			concat(table_schema,".",table_name)
		FROM information_schema.tables 
		WHERE TABLE_SCHEMA = ?
			and table_rows != 0
			and table_rows <= ?
			and TABLE_TYPE = 'BASE TABLE'
		order by table_rows desc
	`
	
	dbData, err := dbObj.Query(getSchema, d, i)
	if err != nil {
		return dbs,err
	}
	defer dbData.Close()

	for dbData.Next() {
		var db string 
		err := dbData.Scan(&db)
		if err != nil {
			return dbs,err
		}

		dbs = append(dbs,db)
	}

	return dbs, nil
}

func Warming(dbObj *sql.DB, ch <-chan string) {
	getCount := "SELECT count(*) FROM %s"
	for v := range ch {
		var cnt int64
		startPnt := time.Now()
		err := dbObj.QueryRow(fmt.Sprintf(getCount,v)).Scan(&cnt)
		if err != nil {
			return
		}
		endPnt := time.Now()
		diff := endPnt.Sub(startPnt)

		
		Print(fmt.Sprintf("T:%s\nC:%d\nD:%s\n====================================",v,cnt,diff))
		
	}
}