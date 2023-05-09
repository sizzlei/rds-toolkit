package lib

import (
	log "github.com/sirupsen/logrus"
	"fmt"
	"errors"
	"os"
)

type Config struct {
	Snapshot	SnapshotConfigure 	`yaml:"snapshot"`
	Upgrade 	[]UpgradeDivInfo	`yaml:"upgrade"`
}

func GetOpt(msg string) (*string, error){
	var x string 
	fmt.Printf("%s : ",msg)
	_, _ = fmt.Scanf("%s",&x)

	log.Infof("%s : %s",msg,x)

	if len(x) == 0 {
		return nil, errors.New(fmt.Sprintf("Not %s Args.",msg))
	}

	return &x, nil
}

func Exiter(err error) {
	fmt.Println(err)
	log.Error(err)
	os.Exit(1)
}

func Print(s string) {
	fmt.Println(s)
	log.Infof(s)
}