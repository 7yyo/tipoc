package mysql

import (
	"fmt"
)

const ShowPlacementLabels = "show placement labels;"

func Count(table string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM %s;", table)
}

func ShowCreateTable(table string) string {
	return fmt.Sprintf("SHOW CREATE TABLE %s;", table)
}

func AddIndex(table, index string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD INDEX %s;", table, index)
}

func ModifyColumn(table, col string) string {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s;", table, col)
}

func CreatePlacementPolicy(p string) string {
	return fmt.Sprintf("CREATE PLACEMENT POLICY %s;", p)
}

func DropPlacementPolicy(p string) string {
	return fmt.Sprintf("DROP PLACEMENT POLICY IF EXISTS %s;", p)
}

func AlterPlacementPolicy(table, p string) string {
	return fmt.Sprintf("ALTER TABLE %s PLACEMENT POLICY = %s;", table, p)
}

func ImportInto(table, csv string) string {
	return fmt.Sprintf("IMPORT INTO %s FROM '%s';", table, csv)
}

func LoadData(table, csv string) string {
	return fmt.Sprintf("LOAD DATA LOCAL INFILE '%s' INTO TABLE %s FIELDS TERMINATED BY ',';", csv, table)
}

func SelectInfoFile(table, csv string) string {
	return fmt.Sprintf("SELECT * FROM %s INTO OUTFILE '%s' FIELDS TERMINATED BY ',';", table, csv)
}
