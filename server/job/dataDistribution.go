package job

import (
	"fmt"
	"pictorial/log"
	"pictorial/operator"
)

func (j *Job) runDataDistribution() {
	tp := operator.GetOTypeValue(operator.DataDistribution)
	lName := fmt.Sprintf("%s/%s.log", j.resultPath, operator.GetOTypeValue(operator.DataDistribution))
	log.Logger.Infof("[%s] start load: %s", tp, Ld.Cmd)
	go Ld.captureLoadLog(lName, j.Channel.ErrC, j.Channel.LdC)
	Ld.run(lName, j.Channel.ErrC, nil)
	j.BarC <- 1
}
