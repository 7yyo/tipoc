package job

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"pictorial/log"
	"pictorial/mysql"
	"strings"
)

const rootUser = "## root"
const tidbUser = "## tidb_user"
const tidbUserName = "tidb_user"
const tidbUserPassword = "tidb_password"
const tidbUserWrongPassword = "tidb_wrong_password"

func (j *Job) runSafety() {
	var cnt int
	j.selected.Walk(func(i *widgets.TreeNode) bool {
		name := i.Value.String()
		scripts := j.examples[i.Value.String()]
		var err error
		var output []string
		var errOutput string
		for i, sql := range scripts {
			user := strings.Split(sql, "\n")[0]
			sql = strings.Trim(sql, "\n")
			switch {
			case strings.Contains(user, rootUser):
				sql = strings.Trim(sql, fmt.Sprintf("%s\n", rootUser))
				output, err = mysql.M.ExecuteForceWithOutput(sql, mysql.M.User, mysql.M.Password)
			case strings.Contains(user, tidbUser):
				sql = strings.Trim(sql, fmt.Sprintf("%s\n", tidbUser))
				if isLoginFailureLimit(name) {
					output, err = mysql.M.ExecuteForceWithOutput(sql, tidbUserName, tidbUserWrongPassword)
				} else {
					output, err = mysql.M.ExecuteForceWithOutput(sql, tidbUserName, tidbUserPassword)
				}
			default:
				err = fmt.Errorf("invalid username, please use 'root' and 'tidb_user'")
			}
			if err != nil {
				errOutput = err.Error()
			}
			j.writeResultFile(name, len(scripts), i, output)
		}
		if errOutput != "" {
			log.Logger.Infof("[warn] %s: %s", name, errOutput)
		} else {
			log.Logger.Infof("[pass] %s", name)
		}
		cnt++
		j.Channel.BarC <- cnt
		return true
	})

}

func isLoginFailureLimit(v string) bool {
	return strings.HasPrefix(v, "6.4")
}
