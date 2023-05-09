package lib 

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"fmt"
)


func GetDBObject(e string,ep int, u string, p string,d string) (*sql.DB, error){
	// e = endpoint, ep = port, u = user,p = password, d = database
	var dbObj *sql.DB
	DSN := "%s:%s@tcp(%s:%d)/%s"
	// Create DB Object
	dbObj, err := sql.Open("mysql",fmt.Sprintf(DSN,u,p,e,ep,d))
	if err != nil {
		return dbObj,err
	}

	var result int
	err = dbObj.QueryRow("select 1").Scan(&result)
	if err != nil {
		return dbObj,err
	}

	return dbObj,nil
}

// PostgreSQL
func GetPostObject(e string,ep int, u string, p string,d string) (*sql.DB, error) {
	// DSN := "postgres://%s:%s@%s:%d/%s?sslmode=require"
	DSN := "host=%s port=%d user=%s password=%s dbname=%s sslmode=require"
	dbObj, err := sql.Open("postgres",
		// fmt.Sprintf(DSN,dbInfo.User,dbInfo.Password,dbInfo.Endpoint,dbInfo.Port,dbInfo.Db),
		fmt.Sprintf(DSN,e,ep,u,p,d),
	)
	if err != nil {
		return nil, err
	}

	var result int
	err = dbObj.QueryRow("select 1").Scan(&result)
	if err != nil {
		return dbObj,err
	}

	return dbObj, nil
}