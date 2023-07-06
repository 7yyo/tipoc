package mysql

import (
	"fmt"
	"github.com/google/uuid"
	"os"
	"pictorial/log"
	"strings"
)

func CreateTableSQL(tblName string, colNum int) strings.Builder {
	sql := strings.Builder{}
	sql.WriteString(fmt.Sprintf("create table poc.%s (id int primary key, c1 varchar(11)", tblName))
	for i := 2; i <= colNum; i++ {
		sql.WriteString(fmt.Sprintf(" , c%d varchar(11)", i))
	}
	sql.WriteString(");")
	return sql
}

func RandomStr(length int) string {
	u, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	return u.String()[:length]
}

func InitCSV(fName string, rowSize, colNum int) error {
	log.Logger.Infof("init csv start: %s", fName)
	csv, err := os.Create(fName)
	if err != nil {
		return err
	}
	var colInfo strings.Builder
	colInfo.WriteString("%d")
	for i := 0; i < colNum; i++ {
		colInfo.WriteString(",%s")
	}
	colInfo.WriteString("\n")
	for i := 1; i <= rowSize; i++ {
		line := fmt.Sprintf(colInfo.String(), i,
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
			RandomStr(11),
		)
		_, _ = csv.WriteString(line)
	}
	log.Logger.Infof("init csv complete: %s", fName)
	return nil
}
