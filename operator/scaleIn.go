package operator

import (
	"fmt"
	"pictorial/log"
	"pictorial/ssh"
)

type scaleInOperator struct {
	host        string
	port        string
	clusterName string
}

func (s *scaleInOperator) Execute() error {
	c := fmt.Sprintf("tiup cluster scale-in %s -N %s:%s -o %s --yes", s.clusterName, s.host, s.port, ssh.S.PrivateKey)
	o, err := ssh.RunLocal(c)
	if err != nil {
		return fmt.Errorf("%s failed, err: %w: %s", c, err, o)
	}
	log.Logger.Info("scaling: %s", string(o))
	return nil
}
