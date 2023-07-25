package bench

import (
	"fmt"
	"pictorial/mysql"
	"pictorial/ssh"
)

type Tpcc struct {
	Mysql      mysql.MySQL
	DB         string
	Warehouses int
	Threads    int
	Cmd        string
}

func (t *Tpcc) String() string {
	return fmt.Sprintf("tiup bench tpcc --host %s --port %s --user %s --password '%s' --warehouses %d --threads %d %s;",
		mysql.M.Host,
		mysql.M.Port,
		mysql.M.User,
		mysql.M.Password,
		t.Warehouses,
		t.Threads,
		t.Cmd)
}

func (t *Tpcc) Run() error {
	if _, err := ssh.S.RunLocal(t.String()); err != nil {
		return err
	}
	return nil
}
